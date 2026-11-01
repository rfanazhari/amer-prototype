package schema_test

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/golang-migrate/migrate/v4/database"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/rfanazhari/amer-prototype/infrastructure/database/schema"
	"github.com/stretchr/testify/require"
)

// migrationCase describes an independently specified adoption result.
type migrationCase struct {
	name    string
	setup   string
	action  string
	version uint
	want    int
	wantErr bool
}

// TestPostgresMigration exercises only an explicitly authorized disposable migration database.
func TestPostgresMigration(t *testing.T) {
	url := os.Getenv("TEST_MIGRATION_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_MIGRATION_DATABASE_URL absent; real migration proof skipped")
	}
	db, err := sql.Open("pgx", url)
	require.NoError(t, err)
	defer func() { _ = db.Close() }()
	ctx := context.Background()
	legacy, err := os.ReadFile("../../../test/integration/testdata/schema/legacy_automigrate.sql")
	require.NoError(t, err)
	directory, err := filepath.Abs("../../../migrations/pgsql")
	require.NoError(t, err)
	runner := schema.Runner{DB: db, Directory: directory}
	cases := []migrationCase{
		{"fresh", "", "up", 2, 2, false},
		{"full legacy", string(legacy), "baseline", 0, 2, false},
		{"task only", string(legacy) + "DROP TABLE administrative_sessions; DROP TABLE administrative_accounts;", "baseline", 0, 1, false},
		{"partial", string(legacy) + "DROP TABLE administrative_sessions;", "baseline", 0, 0, true},
		{"column drift", string(legacy) + "ALTER TABLE tasks ALTER COLUMN title DROP NOT NULL;", "baseline", 0, 0, true},
		{"index drift", string(legacy) + `DROP INDEX "administrative_accounts_email_INDEX";`, "baseline", 0, 0, true},
		{"fk drift", string(legacy) + "ALTER TABLE administrative_sessions DROP CONSTRAINT administrative_sessions_account_id_fkey;", "baseline", 0, 0, true},
		{"foreign metadata", "CREATE TABLE schema_migrations(x text);", "baseline", 0, 0, true},
		{"dirty", "CREATE TABLE schema_migrations(version bigint PRIMARY KEY, dirty boolean NOT NULL); INSERT INTO schema_migrations VALUES(1,true);", "up", 2, 0, true},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			_, e := db.ExecContext(ctx, "DROP SCHEMA public CASCADE; CREATE SCHEMA public;")
			require.NoError(t, e)
			if tt.setup != "" {
				_, e = db.ExecContext(ctx, tt.setup)
				require.NoError(t, e)
			}
			v, d, e := runner.Execute(ctx, schema.Options{Action: tt.action, Version: tt.version})
			if tt.wantErr {
				require.Error(t, e)
				return
			}
			require.NoError(t, e)
			require.Equal(t, tt.want, v)
			require.False(t, d)

			if tt.want == 2 {
				q, e := os.ReadFile("catalog.sql")
				require.NoError(t, e)
				for asset, tables := range map[string][]string{"tasks.json": {"tasks"}, "administrative.json": {"administrative_accounts", "administrative_sessions"}} {
					expected, e := os.ReadFile(asset)
					require.NoError(t, e)
					var actual string
					require.NoError(t, db.QueryRowContext(ctx, string(q), tables).Scan(&actual))
					require.JSONEq(t, string(expected), actual)
				}
			}
			v, d, e = runner.Execute(ctx, schema.Options{Action: "up", Version: 2})
			require.NoError(t, e)
			require.Equal(t, 2, v)
			require.False(t, d)
			_, _, e = runner.Execute(ctx, schema.Options{Action: "up", Version: 2})
			require.NoError(t, e)
			_, _, e = runner.Execute(ctx, schema.Options{Action: "down", Steps: 2, Disposable: true})
			require.NoError(t, e)
			_, _, e = runner.Execute(ctx, schema.Options{Action: "up", Version: 2})
			require.NoError(t, e)
			var count int
			require.NoError(t, db.QueryRowContext(ctx, "SELECT count(*) FROM administrative_accounts").Scan(&count))
			require.Zero(t, count)
		})
	}

	t.Run("immutable artifacts", func(t *testing.T) {
		_, e := db.ExecContext(ctx, "DROP SCHEMA public CASCADE; CREATE SCHEMA public;")
		require.NoError(t, e)
		directoryCopy := t.TempDir()
		for _, name := range []string{"000001_tasks.up.sql", "000001_tasks.down.sql", "000002_administrative_access.up.sql", "000002_administrative_access.down.sql"} {
			b, e := os.ReadFile(filepath.Join(directory, name))
			require.NoError(t, e)
			require.NoError(t, os.WriteFile(filepath.Join(directoryCopy, name), b, 0600))
		}
		require.NoError(t, os.WriteFile(filepath.Join(directoryCopy, "000001_tasks.up.sql"), []byte("SELECT 1;"), 0600))
		_, _, e = (schema.Runner{DB: db, Directory: directoryCopy}).Execute(ctx, schema.Options{Action: "up", Version: 2})
		require.Error(t, e)
		var absent bool
		require.NoError(t, db.QueryRowContext(ctx, "SELECT to_regclass('public.tasks') IS NULL").Scan(&absent))
		require.True(t, absent)
	})
	t.Run("ddl failure retains dirty metadata", func(t *testing.T) {
		_, e := db.ExecContext(ctx, "DROP SCHEMA public CASCADE; CREATE SCHEMA public; CREATE TABLE tasks(id uuid);")
		require.NoError(t, e)
		_, _, e = runner.Execute(ctx, schema.Options{Action: "up", Version: 2})
		require.Error(t, e)
		var dirty bool
		require.NoError(t, db.QueryRowContext(ctx, "SELECT dirty FROM schema_migrations").Scan(&dirty))
		require.True(t, dirty)
		var absent bool
		require.NoError(t, db.QueryRowContext(ctx, "SELECT to_regclass('public.administrative_accounts') IS NULL").Scan(&absent))
		require.True(t, absent)
	})
	t.Run("cancel blocked ddl", func(t *testing.T) {
		_, e := db.ExecContext(ctx, "DROP SCHEMA public CASCADE; CREATE SCHEMA public;")
		require.NoError(t, e)
		_, _, e = runner.Execute(ctx, schema.Options{Action: "up", Version: 2})
		require.NoError(t, e)
		tx, e := db.BeginTx(ctx, nil)
		require.NoError(t, e)
		defer func() { _ = tx.Rollback() }()
		_, e = tx.ExecContext(ctx, "LOCK TABLE administrative_sessions IN ACCESS EXCLUSIVE MODE")
		require.NoError(t, e)
		c, cancel := context.WithTimeout(ctx, 100*time.Millisecond)
		defer cancel()
		started := time.Now()
		_, _, e = runner.Execute(c, schema.Options{Action: "down", Steps: 1, Disposable: true})
		require.Error(t, e)
		require.Less(t, time.Since(started), 2*time.Second)
	})
	t.Run("cancelled operation", func(t *testing.T) {
		c, cancel := context.WithCancel(ctx)
		cancel()
		_, _, e := runner.Execute(c, schema.Options{Action: "up", Version: 2})
		require.Error(t, e)
	})
	t.Run("lock wait", func(t *testing.T) {
		// A concurrent baseline holds the exact same tool lock through comparison and stamping.
		conn, e := db.Conn(ctx)
		require.NoError(t, e)
		defer func() { _ = conn.Close() }()
		var name string
		require.NoError(t, conn.QueryRowContext(ctx, "SELECT current_database()").Scan(&name))
		// The tool-compatible key is generated by the selected dependency, not a second application lock.
		id, e := database.GenerateAdvisoryLockId(name, "public", "schema_migrations")
		require.NoError(t, e)
		_, e = conn.ExecContext(ctx, "SELECT pg_advisory_lock($1)", id)
		require.NoError(t, e)
		defer func() { _, _ = conn.ExecContext(ctx, "SELECT pg_advisory_unlock($1)", id) }()
		started := time.Now()
		_, _, e = runner.Execute(ctx, schema.Options{Action: "up", Version: 2})
		require.Error(t, e)
		require.Less(t, time.Since(started), 8*time.Second)
	})
}
