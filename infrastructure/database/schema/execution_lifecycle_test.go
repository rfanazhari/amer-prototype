package schema_test

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/rfanazhari/amer-prototype/infrastructure/database/schema"
	"github.com/stretchr/testify/require"
)

// lifecycleTables are the five shared tables version 8 adds, in dependency order.
var lifecycleTables = []string{
	"agent_executions",
	"agent_execution_commands",
	"agent_execution_operations",
	"agent_execution_attempts",
	"agent_execution_transitions",
}

// lifecycleUp and lifecycleDown are the version 8 file names.
const (
	lifecycleUp   = "000008_execution_lifecycle.up.sql"
	lifecycleDown = "000008_execution_lifecycle.down.sql"
)

// fixtureIDs are the synthetic identifiers shared by the PostgreSQL scenarios.
const (
	fixtureAccount    = "00000000-0000-0000-0000-0000000000a1"
	fixtureInstance   = "00000000-0000-0000-0000-0000000000b1"
	fixtureValidation = "00000000-0000-0000-0000-0000000000c1"
	fixtureRevision   = "00000000-0000-0000-0000-0000000000d1"
	fixtureCandidate  = "00000000-0000-0000-0000-0000000000e1"
	fixtureAssignment = "00000000-0000-0000-0000-0000000000f1"
	fixturePin        = "00000000-0000-0000-0000-000000000091"
)

// readRepoFile reads a repository file relative to the repository root.
func readRepoFile(t *testing.T, elements ...string) string {
	t.Helper()
	body, err := os.ReadFile(filepath.Join(append([]string{"..", "..", ".."}, elements...)...))
	require.NoError(t, err)
	return string(body)
}

// TestExecutionLifecycleMigrationArtifacts proves the version 8 SQL shape, ordering and scope.
func TestExecutionLifecycleMigrationArtifacts(t *testing.T) {
	up := readRepoFile(t, "migrations", "pgsql", lifecycleUp)
	down := readRepoFile(t, "migrations", "pgsql", lifecycleDown)

	for _, table := range lifecycleTables {
		require.Contains(t, up, "CREATE TABLE public."+table+" (", table)
	}
	for _, snippet := range []string{
		"UNIQUE (scope, command_kind, request_key)",
		"PRIMARY KEY (execution_id, operation_key)",
		"UNIQUE (execution_id, operation_key, attempt_number)",
		"UNIQUE (execution_id, operation_key, authority_generation)",
		"'CREATED'", "'RUNNING'", "'COMPLETED'", "'FAILED'", "'CANCELLED'",
		"state_version >= 1",
		"updated_at >= created_at",
		"committed_progress_ref uuid",
		"REFERENCES public.agent_instances(id) ON UPDATE RESTRICT ON DELETE RESTRICT",
		"REFERENCES public.agent_revisions(id) ON UPDATE RESTRICT ON DELETE RESTRICT",
		"ADD CONSTRAINT interview_executions_id_fkey FOREIGN KEY (id) REFERENCES public.agent_executions(id) ON UPDATE RESTRICT ON DELETE RESTRICT",
	} {
		require.Contains(t, up, snippet)
	}
	for _, forbidden := range []string{"WAITING_INPUT", "ACTIVATING", "RECOVERING", "ADD COLUMN", "DROP COLUMN", "AutoMigrate"} {
		require.NotContains(t, up, forbidden)
	}
	require.NotRegexp(t, regexp.MustCompile(`(?i)stage|invoice|transcript|salary|assignment_id`), up, "kernel tables stay free of interview and invoice facts")
	require.NotRegexp(t, regexp.MustCompile(`ALTER TABLE[^;]*interview_executions\s+ADD COLUMN`), up)

	backfill := strings.Index(up, "INSERT INTO public.agent_executions")
	constraint := strings.Index(up, "ADD CONSTRAINT interview_executions_id_fkey")
	require.GreaterOrEqual(t, backfill, 0, "backfill must exist")
	require.Greater(t, constraint, backfill, "the foreign key follows the backfill")
	require.Equal(t, 1, strings.Count(up, "INSERT INTO"), "only the pin backfill inserts rows; no seed data")
	require.Contains(t, up[backfill:constraint], "FROM public.interview_executions")

	fk := strings.Index(down, "DROP CONSTRAINT IF EXISTS interview_executions_id_fkey")
	require.GreaterOrEqual(t, fk, 0)
	last := fk
	for i := len(lifecycleTables) - 1; i >= 0; i-- {
		at := strings.Index(down, "DROP TABLE IF EXISTS public."+lifecycleTables[i])
		require.Greater(t, at, last, lifecycleTables[i]+" drops after its dependents")
		last = at
	}
	require.NotRegexp(t, regexp.MustCompile(`DROP TABLE[^;]*public\.interview_executions\b`), down, "the pin table is not dropped")
}

// TestExecutionLifecycleChecksums proves the new files are pinned and version 7 is untouched.
func TestExecutionLifecycleChecksums(t *testing.T) {
	var sums map[string]string
	require.NoError(t, json.Unmarshal([]byte(readRepoFile(t, "infrastructure", "database", "schema", "checksums.json")), &sums))
	manifest := readRepoFile(t, "docs", "design", "database-migration-checksums.sha256")
	for _, name := range []string{lifecycleUp, lifecycleDown} {
		body := []byte(readRepoFile(t, "migrations", "pgsql", name))
		digest := sha256.Sum256(body)
		want := hex.EncodeToString(digest[:])
		require.Equal(t, want, sums[name], name)
		require.Contains(t, manifest, want+"  migrations/pgsql/"+name)
	}
	require.Equal(t, "fad47701239f7dd8ecf4d060904a19697ae177dcea39a0508ba85f96825b4b78", sums["000007_interview_executions.up.sql"])
	require.Equal(t, "55a758f53320f47737a270a30a43d0ac9ddf20d760f886e177cbdd9c72304909", sums["000007_interview_executions.down.sql"])
}

// TestReleaseVersionRequiresEight proves the clean schema floor and its operator messages.
func TestReleaseVersionRequiresEight(t *testing.T) {
	require.Equal(t, 8, schema.ReleaseVersion)
	require.Subset(t, schema.RuntimeGrantTables, lifecycleTables)

	tests := []struct {
		name string
		rows *sqlmock.Rows
		err  error
		want string
	}{
		{name: "unreadable", err: errors.New("private"), want: "clean version 8"},
		{name: "old", rows: sqlmock.NewRows([]string{"version", "dirty"}).AddRow(7, false), want: "require clean version 8"},
		{name: "ambiguous", rows: sqlmock.NewRows([]string{"version", "dirty"}).AddRow(8, false).AddRow(9, false), want: "exactly one clean version 8"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			require.NoError(t, err)
			defer func() { _ = db.Close() }()
			expect := mock.ExpectQuery("SELECT version, dirty FROM public.schema_migrations")
			if tt.err != nil {
				expect.WillReturnError(tt.err)
			} else {
				expect.WillReturnRows(tt.rows)
			}
			err = schema.RequireCompatible(context.Background(), db)
			require.ErrorContains(t, err, tt.want)
		})
	}

	runner := schema.Runner{}
	_, _, err := runner.Execute(context.Background(), schema.Options{Action: "up", Version: 9})
	require.ErrorContains(t, err, "1 through 8")
	_, _, err = runner.Execute(context.Background(), schema.Options{Action: "up", Version: 8})
	require.EqualError(t, err, "database required")
	_, _, err = runner.Execute(context.Background(), schema.Options{Action: "down", Steps: 8, Disposable: true})
	require.EqualError(t, err, "database required")
}

// TestOperationalFilesRequireVersionEight proves operator-facing files move to the same floor.
func TestOperationalFilesRequireVersionEight(t *testing.T) {
	makefile := readRepoFile(t, "Makefile")
	require.Contains(t, makefile, "version=8 dirty=false")
	require.Contains(t, makefile, "make docker.migrate MIGRATION_VERSION=8")
	require.NotContains(t, makefile, "version=7 dirty=false")
	require.NotContains(t, makefile, "MIGRATION_VERSION=7")

	compose := readRepoFile(t, "scripts", "verify_compose.py")
	require.Contains(t, compose, "MIGRATION_VERSION=8")
	require.NotContains(t, compose, "MIGRATION_VERSION=7")
	require.NotContains(t, compose, `"MIGRATION_VERSION"] = "7"`)
	require.NotContains(t, compose, "retained version 7")

	readme := readRepoFile(t, "README.md")
	require.Contains(t, readme, "clean version 8")
	require.Contains(t, readme, "MIGRATION_VERSION=8")
	require.NotContains(t, readme, "clean version 7")
	require.NotContains(t, readme, "MIGRATION_VERSION=7")
	require.NotContains(t, readme, "1 through 7")

	runbook := readRepoFile(t, "docs", "design", "database-migrations.md")
	require.Contains(t, runbook, "clean version 8 only")
	require.Contains(t, runbook, "000008")
	require.NotContains(t, runbook, "MIGRATION_VERSION=7")
	require.NotContains(t, runbook, "version=7")
}

// openMigrationDatabase opens the explicitly authorized disposable database or skips.
func openMigrationDatabase(t *testing.T) (*sql.DB, schema.Runner) {
	t.Helper()
	target := os.Getenv("TEST_MIGRATION_DATABASE_URL")
	if target == "" {
		t.Skip("TEST_MIGRATION_DATABASE_URL absent; execution lifecycle migration proof skipped")
	}
	db, err := sql.Open("pgx", target)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, db.Close()) })
	directory, err := filepath.Abs("../../../migrations/pgsql")
	require.NoError(t, err)
	return db, schema.Runner{DB: db, Directory: directory}
}

// resetSchema recreates an empty public schema in the disposable database.
func resetSchema(t *testing.T, db *sql.DB) {
	t.Helper()
	_, err := db.ExecContext(context.Background(), "DROP SCHEMA public CASCADE; CREATE SCHEMA public;")
	require.NoError(t, err)
}

// migrateTo applies the up action and requires the exact clean version.
func migrateTo(t *testing.T, runner schema.Runner, version int) {
	t.Helper()
	got, dirty, err := runner.Execute(context.Background(), schema.Options{Action: "up", Version: uint(version)})
	require.NoError(t, err)
	require.Equal(t, version, got)
	require.False(t, dirty)
}

// seedVersionSevenPin inserts one synthetic assignment pin using only version 7 tables.
func seedVersionSevenPin(t *testing.T, db *sql.DB) {
	t.Helper()
	_, err := db.ExecContext(context.Background(), `
INSERT INTO administrative_accounts(id,name,email,password_hash,role,status,created_at,updated_at) VALUES
('`+fixtureAccount+`','Synthetic Admin','synthetic-admin@example.com','synthetic-hash','ADMIN','ACTIVE','2026-01-01Z','2026-01-01Z');
INSERT INTO agent_instances(id,name,template_id,template_version,behavior_version,current_draft_version,active_revision_id,created_at,updated_at) VALUES
('`+fixtureInstance+`','Synthetic Instance','hr-interviewer','1','1',1,NULL,'2026-01-01Z','2026-01-01Z');
INSERT INTO agent_drafts(instance_id,version,configuration,created_at,updated_at) VALUES ('`+fixtureInstance+`',1,'{}','2026-01-01Z','2026-01-01Z');
INSERT INTO agent_validations(id,instance_id,draft_version,template_id,template_version,behavior_version,eligible,findings,created_at) VALUES
('`+fixtureValidation+`','`+fixtureInstance+`',1,'hr-interviewer','1','1',true,'[]','2026-01-01Z');
INSERT INTO agent_revisions(id,instance_id,revision_number,source_draft_version,validation_id,template_id,template_version,behavior_version,configuration,created_at) VALUES
('`+fixtureRevision+`','`+fixtureInstance+`',1,1,'`+fixtureValidation+`','hr-interviewer','1','1','{}','2026-01-01Z');
INSERT INTO candidates(id,name,email,created_by,created_at,updated_at) VALUES
('`+fixtureCandidate+`','Synthetic Candidate','synthetic-candidate@example.com','`+fixtureAccount+`','2026-01-01Z','2026-01-01Z');
INSERT INTO interview_assignments(id,candidate_id,instance_id,published_revision_id,created_at,updated_at) VALUES
('`+fixtureAssignment+`','`+fixtureCandidate+`','`+fixtureInstance+`','`+fixtureRevision+`','2026-01-01Z','2026-01-01Z');
INSERT INTO interview_executions(id,assignment_id,instance_id,published_revision_id,created_at) VALUES
('`+fixturePin+`','`+fixtureAssignment+`','`+fixtureInstance+`','`+fixtureRevision+`','2026-01-02T03:04:05.678Z');`)
	require.NoError(t, err)
}

// countRows counts rows in a trusted table name.
func countRows(t *testing.T, db *sql.DB, table string) int {
	t.Helper()
	var n int
	require.NoError(t, db.QueryRowContext(context.Background(), "SELECT count(*) FROM public."+table).Scan(&n))
	return n
}

// requireBackfilledPin asserts the CREATED row derived from the version 7 pin.
func requireBackfilledPin(t *testing.T, db *sql.DB) {
	t.Helper()
	var (
		state                                 string
		version                               int64
		instance, revision                    string
		created, updated                      time.Time
		activated, terminal, progress, reason sql.NullString
	)
	require.NoError(t, db.QueryRowContext(context.Background(), `SELECT state, state_version, instance_id::text, published_revision_id::text,
created_at, updated_at, activated_at::text, terminal_at::text, committed_progress_ref::text, terminal_reason
FROM public.agent_executions WHERE id = $1`, fixturePin).Scan(&state, &version, &instance, &revision, &created, &updated, &activated, &terminal, &progress, &reason))
	require.Equal(t, "CREATED", state)
	require.EqualValues(t, 1, version)
	require.Equal(t, fixtureInstance, instance)
	require.Equal(t, fixtureRevision, revision)
	require.True(t, created.Equal(time.Date(2026, 1, 2, 3, 4, 5, 678000000, time.UTC)), created.String())
	require.True(t, updated.Equal(created))
	for _, value := range []sql.NullString{activated, terminal, progress, reason} {
		require.False(t, value.Valid)
	}
	for _, table := range lifecycleTables[1:] {
		require.Zero(t, countRows(t, db, table), table+" gets no invented history")
	}
}

// TestExecutionLifecycleMigrationOnPostgres proves fresh, upgrade, rerun, constraint and disposable down/up behavior.
func TestExecutionLifecycleMigrationOnPostgres(t *testing.T) {
	db, runner := openMigrationDatabase(t)
	ctx := context.Background()

	t.Run("fresh migrate to 8 has no seed rows", func(t *testing.T) {
		resetSchema(t, db)
		migrateTo(t, runner, 8)
		require.NoError(t, schema.RequireCompatible(ctx, db))
		for _, table := range lifecycleTables {
			require.Zero(t, countRows(t, db, table), table)
		}
		var present bool
		require.NoError(t, db.QueryRowContext(ctx, "SELECT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'interview_executions_id_fkey')").Scan(&present))
		require.True(t, present)
	})

	t.Run("upgrade from 7 backfills, reruns as no-op, and enforces constraints", func(t *testing.T) {
		resetSchema(t, db)
		migrateTo(t, runner, 7)
		seedVersionSevenPin(t, db)
		migrateTo(t, runner, 8)
		requireBackfilledPin(t, db)
		migrateTo(t, runner, 8)
		require.Equal(t, 1, countRows(t, db, "agent_executions"))

		_, err := db.ExecContext(ctx, `INSERT INTO interview_executions(id,assignment_id,instance_id,published_revision_id) VALUES
('00000000-0000-0000-0000-000000000092',$1,$2,$3)`, fixtureAssignment, fixtureInstance, fixtureRevision)
		require.Error(t, err, "a second pin for one assignment is refused")
		const insertExecution = `INSERT INTO agent_executions(id,instance_id,published_revision_id,state,state_version,activated_at,terminal_at,created_at,updated_at)
VALUES ('00000000-0000-0000-0000-000000000093',$1,$2,$3,1,now(),now(),'2026-01-01Z',now())`
		_, err = db.ExecContext(ctx, insertExecution, fixtureInstance, fixtureRevision, "COMPLETED")
		require.ErrorContains(t, err, "violates check constraint", "COMPLETED without progress cannot be stored")
		_, err = db.ExecContext(ctx, insertExecution, fixtureInstance, fixtureRevision, "WAITING_INPUT")
		require.ErrorContains(t, err, "violates check constraint", "only the five states exist")
	})

	t.Run("pin without a shared execution is refused", func(t *testing.T) {
		resetSchema(t, db)
		migrateTo(t, runner, 7)
		seedVersionSevenPin(t, db)
		migrateTo(t, runner, 8)
		_, err := db.ExecContext(ctx, `DELETE FROM public.interview_executions WHERE id = $1`, fixturePin)
		require.NoError(t, err)
		_, err = db.ExecContext(ctx, `INSERT INTO interview_executions(id,assignment_id,instance_id,published_revision_id) VALUES
('00000000-0000-0000-0000-000000000094',$1,$2,$3)`, fixtureAssignment, fixtureInstance, fixtureRevision)
		require.ErrorContains(t, err, "interview_executions_id_fkey")
	})

	t.Run("disposable down keeps the pin and up backfills again", func(t *testing.T) {
		resetSchema(t, db)
		migrateTo(t, runner, 7)
		seedVersionSevenPin(t, db)
		migrateTo(t, runner, 8)
		got, dirty, err := runner.Execute(ctx, schema.Options{Action: "down", Steps: 1, Disposable: true})
		require.NoError(t, err)
		require.Equal(t, 7, got)
		require.False(t, dirty)
		for _, table := range lifecycleTables {
			var absent bool
			require.NoError(t, db.QueryRowContext(ctx, "SELECT to_regclass('public.'||$1) IS NULL", table).Scan(&absent))
			require.True(t, absent, table)
		}
		require.Equal(t, 1, countRows(t, db, "interview_executions"))
		migrateTo(t, runner, 8)
		requireBackfilledPin(t, db)
	})

	t.Run("baseline stamps the catalog it matches", func(t *testing.T) {
		resetSchema(t, db)
		migrateTo(t, runner, 8)
		_, err := db.ExecContext(ctx, "DROP TABLE schema_migrations")
		require.NoError(t, err)
		got, dirty, err := runner.Execute(ctx, schema.Options{Action: "baseline"})
		require.NoError(t, err)
		require.Equal(t, 8, got)
		require.False(t, dirty)

		_, err = db.ExecContext(ctx, "DROP TABLE schema_migrations; DROP TABLE public.agent_execution_transitions")
		require.NoError(t, err)
		_, _, err = runner.Execute(ctx, schema.Options{Action: "baseline"})
		require.Error(t, err, "a partial lifecycle catalog is not adopted")

		resetSchema(t, db)
		migrateTo(t, runner, 7)
		_, err = db.ExecContext(ctx, "DROP TABLE schema_migrations")
		require.NoError(t, err)
		got, _, err = runner.Execute(ctx, schema.Options{Action: "baseline"})
		require.NoError(t, err)
		require.Equal(t, 7, got, "a version 7 catalog stays version 7")
	})
}
