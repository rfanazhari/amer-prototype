package integration_test

import (
	"context"
	"database/sql"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/SALT-Indonesia/salt-pkg/logmanager"
	"github.com/google/uuid"
	"github.com/rfanazhari/amer-prototype/infrastructure/database/schema"
	appruntime "github.com/rfanazhari/amer-prototype/infrastructure/runtime"
	"github.com/stretchr/testify/require"
)

// TestRuntimeRequiresPreparedSchema proves real startup refuses unsafe metadata before binding or bootstrap.
func TestRuntimeRequiresPreparedSchema(t *testing.T) {
	target := os.Getenv("TEST_MIGRATION_DATABASE_URL")
	if target == "" {
		t.Skip("TEST_MIGRATION_DATABASE_URL absent; disposable runtime schema proof skipped")
	}
	db, err := sql.Open("pgx", target)
	require.NoError(t, err)
	defer func() { require.NoError(t, db.Close()) }()
	// startupSchemaCase describes incompatible metadata with an independently specified setup.
	type startupSchemaCase struct {
		name    string
		setup   string
		wantErr bool
	}
	tests := []startupSchemaCase{
		{name: "missing", wantErr: true},
		{name: "unreadable metadata", setup: "CREATE TABLE schema_migrations(x text);", wantErr: true},
		{name: "ambiguous metadata", setup: "CREATE TABLE schema_migrations(version bigint,dirty boolean); INSERT INTO schema_migrations VALUES(2,false),(2,false);", wantErr: true},
		{name: "empty", setup: "CREATE TABLE schema_migrations(version bigint PRIMARY KEY,dirty boolean NOT NULL);", wantErr: true},
		{name: "old", setup: "CREATE TABLE schema_migrations(version bigint PRIMARY KEY,dirty boolean NOT NULL); INSERT INTO schema_migrations VALUES(1,false);", wantErr: true},
		{name: "future", setup: "CREATE TABLE schema_migrations(version bigint PRIMARY KEY,dirty boolean NOT NULL); INSERT INTO schema_migrations VALUES(3,false);", wantErr: true},
		{name: "dirty", setup: "CREATE TABLE schema_migrations(version bigint PRIMARY KEY,dirty boolean NOT NULL); INSERT INTO schema_migrations VALUES(2,true);", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := db.ExecContext(context.Background(), "DROP SCHEMA public CASCADE; CREATE SCHEMA public;"+tt.setup)
			require.NoError(t, err)
			parsed, err := url.Parse(target)
			require.NoError(t, err)
			label := "schema_" + uuid.NewString()
			query := parsed.Query()
			query.Set("application_name", label)
			parsed.RawQuery = query.Encode()
			bound := false
			cfg := lifecycleConfig(parsed.String(), "127.0.0.1:0", nil)
			cfg.Listen = func(context.Context, string, string) (net.Listener, error) { bound = true; return nil, nil }
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()
			err = appruntime.Run(ctx, cfg, logmanager.NewApplication(logmanager.WithAppName("schema-test")))
			require.ErrorContains(t, err, "schema")
			require.False(t, bound)
			var businessTables, connections int
			require.NoError(t, db.QueryRowContext(context.Background(), "SELECT count(*) FROM pg_tables WHERE schemaname='public' AND tablename<>'schema_migrations'").Scan(&businessTables))
			require.Zero(t, businessTables)
			require.NoError(t, db.QueryRowContext(context.Background(), "SELECT count(*) FROM pg_stat_activity WHERE application_name=$1", label).Scan(&connections))
			require.Zero(t, connections)
		})
	}
}

// TestRuntimeWithDDLDeniedRole proves the cutover binary works with CRUD and metadata reads only.
func TestRuntimeWithDDLDeniedRole(t *testing.T) {
	target := os.Getenv("TEST_MIGRATION_DATABASE_URL")
	if target == "" {
		t.Skip("TEST_MIGRATION_DATABASE_URL absent; restricted-role proof skipped")
	}
	db, err := sql.Open("pgx", target)
	require.NoError(t, err)
	defer func() { require.NoError(t, db.Close()) }()
	_, err = db.ExecContext(context.Background(), "DROP SCHEMA public CASCADE; CREATE SCHEMA public;")
	require.NoError(t, err)
	directory, err := filepath.Abs("../../migrations/pgsql")
	require.NoError(t, err)
	_, _, err = (schema.Runner{DB: db, Directory: directory}).Execute(context.Background(), schema.Options{Action: "up", Version: 2})
	require.NoError(t, err)
	role := "runtime_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	password := uuid.NewString()
	// Both interpolated values are generated UUIDs, never external identifiers or credentials.
	_, err = db.ExecContext(context.Background(), "CREATE ROLE "+role+" LOGIN PASSWORD '"+password+"'; GRANT USAGE ON SCHEMA public TO "+role+"; GRANT SELECT,INSERT,UPDATE,DELETE ON tasks,administrative_accounts,administrative_sessions TO "+role+"; GRANT SELECT ON schema_migrations TO "+role)
	require.NoError(t, err)
	defer func() {
		_, err := db.ExecContext(context.Background(), "DROP OWNED BY "+role+"; DROP ROLE "+role)
		require.NoError(t, err)
	}()
	parsed, err := url.Parse(target)
	require.NoError(t, err)
	parsed.User = url.UserPassword(role, password)
	restricted, err := sql.Open("pgx", parsed.String())
	require.NoError(t, err)
	defer func() { require.NoError(t, restricted.Close()) }()
	for _, statement := range []string{"CREATE TABLE forbidden(id int)", "ALTER TABLE tasks ADD COLUMN forbidden int", "DROP TABLE tasks", "CREATE TABLE schema_migrations_other(version int)", "UPDATE schema_migrations SET dirty=true"} {
		_, err = restricted.ExecContext(context.Background(), statement)
		require.Error(t, err)
	}
	require.NoError(t, schema.RequireCompatible(context.Background(), restricted))
	for i := 0; i < 2; i++ {
		ready := make(chan net.Addr, 1)
		result := make(chan error, 1)
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		cfg := lifecycleConfig(parsed.String(), "127.0.0.1:0", ready)
		go func() {
			result <- appruntime.Run(ctx, cfg, logmanager.NewApplication(logmanager.WithAppName("restricted-runtime")))
		}()
		select {
		case addr := <-ready:
			client := http.Client{Timeout: time.Second}
			response, err := client.Get("http://" + addr.String() + "/health")
			require.NoError(t, err)
			require.Equal(t, http.StatusOK, response.StatusCode)
			require.NoError(t, response.Body.Close())
		case err := <-result:
			cancel()
			t.Fatalf("restricted startup failed: %v", err)
		case <-time.After(10 * time.Second):
			cancel()
			t.Fatal("restricted startup timed out")
		}
		cancel()
		select {
		case err := <-result:
			require.NoError(t, err)
		case <-time.After(6 * time.Second):
			t.Fatal("restricted shutdown exceeded budget")
		}
	}
}
