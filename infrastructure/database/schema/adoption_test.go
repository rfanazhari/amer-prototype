package schema_test

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/rfanazhari/amer-prototype/application/service"
	"github.com/rfanazhari/amer-prototype/infrastructure/database/schema"
	"github.com/rfanazhari/amer-prototype/infrastructure/repository/accessrepository"
	"github.com/rfanazhari/amer-prototype/infrastructure/service/accessauthority"
	"github.com/rfanazhari/amer-prototype/infrastructure/service/accesscredentials"
	"github.com/rfanazhari/amer-prototype/infrastructure/service/clock"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// proofDatabase opens only the explicitly authorized disposable migration database.
func proofDatabase(t *testing.T) (*sql.DB, schema.Runner) {
	t.Helper()
	target := os.Getenv("TEST_MIGRATION_DATABASE_URL")
	if target == "" {
		t.Skip("TEST_MIGRATION_DATABASE_URL absent; migration acceptance skipped")
	}
	db, err := sql.Open("pgx", target)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, db.Close()) })
	directory, err := filepath.Abs("../../../migrations/pgsql")
	require.NoError(t, err)
	return db, schema.Runner{DB: db, Directory: directory}
}

// resetProof clears only the disposable schema before one independently arranged proof.
func resetProof(t *testing.T, db *sql.DB, setup string) {
	t.Helper()
	_, err := db.ExecContext(context.Background(), "DROP SCHEMA public CASCADE; CREATE SCHEMA public;"+setup)
	require.NoError(t, err)
}

// retainedSnapshot reads complete row facts into memory; comparisons never print their contents.
func retainedSnapshot(t *testing.T, db *sql.DB, full bool) string {
	t.Helper()
	query := `SELECT jsonb_build_object('tasks', (SELECT jsonb_agg(to_jsonb(t) ORDER BY id) FROM tasks t))::text`
	if full {
		query = `SELECT jsonb_build_object('tasks', (SELECT jsonb_agg(to_jsonb(t) ORDER BY id) FROM tasks t), 'accounts', (SELECT jsonb_agg(to_jsonb(a) ORDER BY id) FROM administrative_accounts a), 'sessions', (SELECT jsonb_agg(to_jsonb(s) ORDER BY id) FROM administrative_sessions s))::text`
	}
	var snapshot string
	require.NoError(t, db.QueryRowContext(context.Background(), query).Scan(&snapshot))
	return snapshot
}

// TestPopulatedAdoptionPreservesEveryStoredFact verifies both supported populated legacy shapes.
func TestPopulatedAdoptionPreservesEveryStoredFact(t *testing.T) {
	db, runner := proofDatabase(t)
	legacy, err := os.ReadFile("../../../test/integration/testdata/schema/legacy_automigrate.sql")
	require.NoError(t, err)
	// adoptionCase selects the supported legacy schema independently of runner internals.
	type adoptionCase struct {
		name string
		full bool
		want int
	}
	for _, tt := range []adoptionCase{{name: "task only", want: 1}, {name: "full legacy", full: true, want: 2}} {
		t.Run(tt.name, func(t *testing.T) {
			resetProof(t, db, string(legacy))
			_, err := db.ExecContext(context.Background(), `INSERT INTO tasks(id,title,status,due_date,completed_at,created_at,updated_at) VALUES
    ('00000000-0000-0000-0000-000000000001','retained completed task','done','2026-01-02Z','2026-01-01T01:00:00.123Z','2026-01-01Z','2026-01-01T01:00:00.123Z');`)
			require.NoError(t, err)
			if tt.full {
				_, err = db.ExecContext(context.Background(), `INSERT INTO administrative_accounts(id,name,email,password_hash,role,status,created_at,updated_at) VALUES
     ('00000000-0000-0000-0000-000000000002','Synthetic Admin','synthetic-admin@example.com','synthetic-hash','ADMIN','ACTIVE','2026-01-01Z','2026-01-01Z');
     INSERT INTO administrative_accounts(id,name,email,password_hash,role,status,created_by,created_at,updated_at) VALUES
     ('00000000-0000-0000-0000-000000000003','Synthetic HR','synthetic-hr@example.com','synthetic-hr-hash','HR','ACTIVE','00000000-0000-0000-0000-000000000002','2026-01-01Z','2026-01-01Z');
     INSERT INTO administrative_sessions(id,account_id,token_hash,created_at,expires_at,revoked_at) VALUES
     ('00000000-0000-0000-0000-000000000004','00000000-0000-0000-0000-000000000003',repeat('a',64),'2026-01-01Z','2099-01-01Z',NULL),
     ('00000000-0000-0000-0000-000000000005','00000000-0000-0000-0000-000000000003',repeat('b',64),'2026-01-01Z','2099-01-01Z','2026-01-02Z');`)
				require.NoError(t, err)
			} else {
				_, err = db.ExecContext(context.Background(), "DROP TABLE administrative_sessions; DROP TABLE administrative_accounts;")
				require.NoError(t, err)
			}
			live, revoked := "", ""
			if tt.full {
				live = replaceSessionDigest(t, db, "00000000-0000-0000-0000-000000000004")
				revoked = replaceSessionDigest(t, db, "00000000-0000-0000-0000-000000000005")
			}
			before := retainedSnapshot(t, db, tt.full)
			v, dirty, err := runner.Execute(context.Background(), schema.Options{Action: "baseline"})
			require.NoError(t, err)
			require.Equal(t, tt.want, v)
			require.False(t, dirty)
			require.True(t, before == retainedSnapshot(t, db, tt.full), "adoption altered retained facts")
			_, _, err = runner.Execute(context.Background(), schema.Options{Action: "up", Version: 2})
			require.NoError(t, err)
			require.True(t, before == retainedSnapshot(t, db, tt.full), "upgrade altered retained facts")
			if tt.full {
				orm, err := gorm.Open(postgres.New(postgres.Config{Conn: db}), &gorm.Config{DisableAutomaticPing: true, Logger: logger.Default.LogMode(logger.Silent)})
				require.NoError(t, err)
				sessions := accessauthority.Sessions{Store: accessrepository.Store{DB: orm}, Tokens: accesscredentials.Tokens{}, Clock: clock.System{}}
				_, account, err := sessions.Resolve(service.WithAdministrativeCredential(context.Background(), live))
				require.NoError(t, err)
				require.Equal(t, "00000000-0000-0000-0000-000000000003", account.ID)
				_, _, err = sessions.Resolve(service.WithAdministrativeCredential(context.Background(), revoked))
				require.Error(t, err)
			}
		})
	}
}

// TestConcurrentMigratorsSerialize proves competing real commands converge on the release version.
func TestConcurrentMigratorsSerialize(t *testing.T) {
	db, runner := proofDatabase(t)
	resetProof(t, db, "")
	start := make(chan struct{})
	results := make(chan error, 2)
	var workers sync.WaitGroup
	for i := 0; i < 2; i++ {
		workers.Go(func() {
			<-start
			_, _, err := runner.Execute(context.Background(), schema.Options{Action: "up", Version: 2})
			results <- err
		})
	}
	close(start)
	workers.Wait()
	close(results)
	for err := range results {
		require.NoError(t, err)
	}
	require.NoError(t, schema.RequireCompatible(context.Background(), db))
}

// TestMigrationRefusalsAndRecovery proves semantic drift refusal and transaction-safe forward recovery.
func TestMigrationRefusalsAndRecovery(t *testing.T) {
	db, runner := proofDatabase(t)
	legacy, err := os.ReadFile("../../../test/integration/testdata/schema/legacy_automigrate.sql")
	require.NoError(t, err)
	// refusalCase defines an independently injected incompatible catalog.
	type refusalCase struct {
		name    string
		setup   string
		action  string
		wantErr bool
	}
	tests := []refusalCase{
		{name: "wrong index definition", setup: string(legacy) + `DROP INDEX "administrative_accounts_email_INDEX"; CREATE INDEX "administrative_accounts_email_INDEX" ON administrative_accounts(name);`, action: "baseline", wantErr: true},
		{name: "wrong fk action", setup: string(legacy) + `ALTER TABLE administrative_sessions DROP CONSTRAINT administrative_sessions_account_id_fkey; ALTER TABLE administrative_sessions ADD CONSTRAINT administrative_sessions_account_id_fkey FOREIGN KEY(account_id) REFERENCES administrative_accounts(id) ON DELETE CASCADE ON UPDATE RESTRICT;`, action: "baseline", wantErr: true},
		{name: "future metadata", setup: `CREATE TABLE schema_migrations(version bigint PRIMARY KEY,dirty boolean NOT NULL); INSERT INTO schema_migrations VALUES(3,false);`, action: "up", wantErr: true},
		{name: "zero metadata", setup: `CREATE TABLE schema_migrations(version bigint PRIMARY KEY,dirty boolean NOT NULL); INSERT INTO schema_migrations VALUES(0,false);`, action: "up", wantErr: true},
		{name: "ambiguous metadata", setup: `CREATE TABLE schema_migrations(version bigint PRIMARY KEY,dirty boolean NOT NULL); INSERT INTO schema_migrations VALUES(1,false),(2,false);`, action: "up", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetProof(t, db, tt.setup)
			_, _, err := runner.Execute(context.Background(), schema.Options{Action: tt.action, Version: 2})
			require.Error(t, err)
			if tt.action == "baseline" {
				var absent bool
				require.NoError(t, db.QueryRowContext(context.Background(), "SELECT to_regclass('public.schema_migrations') IS NULL").Scan(&absent))
				require.True(t, absent)
			}
		})
	}
	t.Run("cancelled DDL refuses restart until reviewed recovery", func(t *testing.T) {
		resetProof(t, db, "")
		_, _, err := runner.Execute(context.Background(), schema.Options{Action: "up", Version: 2})
		require.NoError(t, err)
		tx, err := db.BeginTx(context.Background(), nil)
		require.NoError(t, err)
		_, err = tx.ExecContext(context.Background(), "LOCK TABLE administrative_sessions IN ACCESS EXCLUSIVE MODE")
		require.NoError(t, err)
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()
		_, _, err = runner.Execute(ctx, schema.Options{Action: "down", Steps: 1, Disposable: true})
		require.Error(t, err)
		// Keep the blocker until pgx's asynchronous wire cancellation has reached PostgreSQL.
		// Releasing it first could let the DDL finish before server-side cancellation.
		require.Eventually(t, func() bool {
			var active int
			err := db.QueryRowContext(context.Background(), "SELECT count(*) FROM pg_stat_activity WHERE datname=current_database() AND pid<>pg_backend_pid() AND state='active'").Scan(&active)
			return err == nil && active == 0
		}, 2*time.Second, 10*time.Millisecond)
		require.NoError(t, tx.Rollback())
		require.Error(t, schema.RequireCompatible(context.Background(), db))
		_, _, err = runner.Execute(context.Background(), schema.Options{Action: "up", Version: 2})
		require.Error(t, err)
		// This injected down was transactional: catalog comparison must establish the actual state before clearing dirty.
		catalog, err := os.ReadFile("catalog.sql")
		require.NoError(t, err)
		expected, err := os.ReadFile("administrative.json")
		require.NoError(t, err)
		var actual string
		require.NoError(t, db.QueryRowContext(context.Background(), string(catalog), []string{"administrative_accounts", "administrative_sessions"}).Scan(&actual))
		require.JSONEq(t, string(expected), actual)
		// golang-migrate records the down target (1) as dirty before running version 2 down.
		repaired, err := db.ExecContext(context.Background(), "UPDATE schema_migrations SET version=2,dirty=false WHERE version=1 AND dirty=true")
		require.NoError(t, err)
		affected, err := repaired.RowsAffected()
		require.NoError(t, err)
		require.Equal(t, int64(1), affected)
		require.NoError(t, schema.RequireCompatible(context.Background(), db))
		_, _, err = runner.Execute(context.Background(), schema.Options{Action: "up", Version: 2})
		require.NoError(t, err)
	})
}

// replaceSessionDigest supplies a synthetic usable credential without printing its plaintext or digest.
func replaceSessionDigest(t *testing.T, db *sql.DB, id string) string {
	t.Helper()
	token, digest, err := (accesscredentials.Tokens{}).Issue(context.Background(), service.CredentialAdministrative)
	require.NoError(t, err)
	_, err = db.ExecContext(context.Background(), "UPDATE administrative_sessions SET token_hash=$1 WHERE id=$2", digest, id)
	require.NoError(t, err)
	return token
}
