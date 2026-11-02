package schema_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"regexp"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/rfanazhari/amer-prototype/infrastructure/database/schema"
	"github.com/stretchr/testify/require"
)

// interviewRuntimeTables are the eight tables version 10 adds, in dependency order.
var interviewRuntimeTables = []string{
	"interview_progress",
	"interview_coverage_items",
	"interview_technical_losses",
	"interview_voice_sessions",
	"interview_transcript_entries",
	"interview_checkpoints",
	"interview_handoffs",
	"interview_session_commands",
}

// interviewRuntimeUp and interviewRuntimeDown are the version 10 file names.
const (
	interviewRuntimeUp   = "000010_interview_runtime.up.sql"
	interviewRuntimeDown = "000010_interview_runtime.down.sql"
)

// TestInterviewRuntimeMigrationArtifacts proves the version 10 SQL shape, scope and drop order.
func TestInterviewRuntimeMigrationArtifacts(t *testing.T) {
	up := readRepoFile(t, "migrations", "pgsql", interviewRuntimeUp)
	down := readRepoFile(t, "migrations", "pgsql", interviewRuntimeDown)

	for _, table := range interviewRuntimeTables {
		require.Contains(t, up, "CREATE TABLE public."+table+" (", table)
	}
	require.Equal(t, 8, strings.Count(up, "CREATE TABLE"), "only the eight interview runtime tables are created")

	for _, snippet := range []string{
		// interview_progress
		"execution_id uuid PRIMARY KEY",
		"follow_up_limit integer NOT NULL",
		"follow_up_limit = 2",
		"current_stage text NOT NULL",
		"'background', 'screening', 'compensation', 'closing'",
		"committed_version bigint NOT NULL",
		"committed_version >= 1",
		"updated_at >= created_at",
		"REFERENCES public.interview_executions(id) ON UPDATE RESTRICT ON DELETE RESTRICT",
		"REFERENCES public.interview_progress(execution_id) ON UPDATE RESTRICT ON DELETE RESTRICT",
		// follow-ups and reconnect counters on progress
		"follow_up_entry_ids jsonb NOT NULL DEFAULT '[]'::jsonb",
		"jsonb_typeof(follow_up_entry_ids) = 'array'",
		"reconnect_episodes integer NOT NULL DEFAULT 0",
		"reconnect_attempts integer NOT NULL DEFAULT 0",
		"reconnect_detected_at timestamptz NULL",
		"reconnect_episodes BETWEEN 0 AND 2",
		"reconnect_attempts BETWEEN 0 AND 2",
		// clarification and omission evidence on coverage
		"clarification_entry_id uuid NULL",
		"omission_entry_id uuid NULL",
		"(clarification_count = 1) = (clarification_entry_id IS NOT NULL)",
		"interview_coverage_items_clarification_entry_id_fkey",
		"interview_coverage_items_omission_entry_id_fkey",
		// coverage
		"UNIQUE (execution_id, question_id)",
		"'pending', 'asked', 'answered', 'omitted'",
		"candidate_skip",
		"no_answer_after_clarification",
		"soft_budget_expired",
		"clarification_count >= 0",
		"clarification_count <= 1",
		"answer_entry_id uuid",
		"REFERENCES public.interview_transcript_entries(id) ON UPDATE RESTRICT ON DELETE RESTRICT",
		// technical losses
		"WHERE resolved_at IS NULL",
		// voice sessions
		"WHERE closed_at IS NULL",
		"WHERE provider_session_id IS NOT NULL",
		// transcript
		"'candidate', 'model'",
		"'fragment', 'final'",
		"UNIQUE (session_id, ordinal, revision)",
		"REFERENCES public.interview_voice_sessions(id) ON UPDATE RESTRICT ON DELETE RESTRICT",
		// checkpoints
		"interview_checkpoints_execution_id_created_at_INDEX",
		// handoffs
		"UNIQUE (execution_id, source_stage)",
		"UNIQUE (execution_id, revision)",
		// session commands
		"'activate', 'reconnect'",
		"UNIQUE (execution_id, idempotency_key)",
	} {
		require.Contains(t, up, snippet, snippet)
	}

	require.NotContains(t, up, "INSERT INTO", "no seed rows")
	for _, forbidden := range []string{"AutoMigrate", "ADD COLUMN", "DROP COLUMN", "ON DELETE CASCADE", "deleted_at"} {
		require.NotContains(t, up, forbidden)
	}
	require.NotContains(t, up, "REFERENCES public.agent_execution_attempts", "attempt_id has no foreign key")
	require.NotRegexp(t, regexp.MustCompile(`(?i)ALTER TABLE[^;]*interview_executions`), up, "the execution pin table gains nothing")

	last := 0
	for i := len(interviewRuntimeTables) - 1; i >= 0; i-- {
		at := strings.Index(down, "DROP TABLE IF EXISTS public."+interviewRuntimeTables[i])
		require.Greater(t, at, last, interviewRuntimeTables[i]+" drops after its dependents")
		last = at
	}
	for _, kept := range []string{"interview_executions", "interview_assignments", "interview_consents", "interview_controllers"} {
		require.NotRegexp(t, regexp.MustCompile(`DROP TABLE[^;]*public\.`+kept+`\b`), down, kept+" is not dropped")
	}
}

// TestInterviewRuntimeChecksums proves the new files are pinned and versions 1 through 9 are untouched.
func TestInterviewRuntimeChecksums(t *testing.T) {
	var sums map[string]string
	require.NoError(t, json.Unmarshal([]byte(readRepoFile(t, "infrastructure", "database", "schema", "checksums.json")), &sums))
	manifest := readRepoFile(t, "docs", "design", "database-migration-checksums.sha256")
	for _, name := range []string{interviewRuntimeUp, interviewRuntimeDown} {
		body := []byte(readRepoFile(t, "migrations", "pgsql", name))
		digest := sha256.Sum256(body)
		want := hex.EncodeToString(digest[:])
		require.Equal(t, want, sums[name], name)
		require.Contains(t, manifest, want+"  migrations/pgsql/"+name)
	}
	require.Len(t, sums, 20, "versions 1 through 10, up and down")
	require.Equal(t, "62358cb4826c1e61a21d863ab0ad0b703f914f728ea03b1ab37c01e1317d8b6c", sums["000009_candidate_interview.up.sql"])
	require.Equal(t, "2ad92faa7bfe6703259da7b01906531f98376b29f02450420c621db088b7f20f", sums["000009_candidate_interview.down.sql"])
}

// TestReleaseVersionRequiresTen proves the clean schema floor, grants, and operator messages.
func TestReleaseVersionRequiresTen(t *testing.T) {
	require.Equal(t, 10, schema.ReleaseVersion)
	require.Subset(t, schema.RuntimeGrantTables, interviewRuntimeTables)
	require.Subset(t, schema.RuntimeGrantTables, candidateInterviewTables, "version 9 grants stay")

	tests := []struct {
		name    string
		rows    *sqlmock.Rows
		err     error
		want    string
		wantErr bool
	}{
		{name: "unreadable", err: errors.New("private"), want: "clean version 10", wantErr: true},
		{name: "floor nine is refused", rows: sqlmock.NewRows([]string{"version", "dirty"}).AddRow(9, false), want: "require clean version 10", wantErr: true},
		{name: "future is refused", rows: sqlmock.NewRows([]string{"version", "dirty"}).AddRow(11, false), want: "require clean version 10", wantErr: true},
		{name: "dirty is refused", rows: sqlmock.NewRows([]string{"version", "dirty"}).AddRow(10, true), want: "require clean version 10", wantErr: true},
		{name: "ambiguous", rows: sqlmock.NewRows([]string{"version", "dirty"}).AddRow(10, false).AddRow(10, false), want: "exactly one clean version 10", wantErr: true},
		{name: "clean ten", rows: sqlmock.NewRows([]string{"version", "dirty"}).AddRow(10, false)},
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
			if !tt.wantErr {
				require.NoError(t, err)
				return
			}
			require.ErrorContains(t, err, tt.want)
		})
	}

	runner := schema.Runner{}
	_, _, err := runner.Execute(context.Background(), schema.Options{Action: "up", Version: 11})
	require.ErrorContains(t, err, "1 through 10")
	_, _, err = runner.Execute(context.Background(), schema.Options{Action: "up", Version: 10})
	require.EqualError(t, err, "database required")
	_, _, err = runner.Execute(context.Background(), schema.Options{Action: "down", Steps: 10, Disposable: true})
	require.EqualError(t, err, "database required")
}

// TestOperationalFilesRequireVersionTen proves operator-facing files move to the same floor.
func TestOperationalFilesRequireVersionTen(t *testing.T) {
	makefile := readRepoFile(t, "Makefile")
	require.Contains(t, makefile, "version=10 dirty=false")
	require.Contains(t, makefile, "make docker.migrate MIGRATION_VERSION=10")
	require.NotContains(t, makefile, "version=9 dirty=false")
	require.NotContains(t, makefile, "MIGRATION_VERSION=9")

	compose := readRepoFile(t, "scripts", "verify_compose.py")
	require.Contains(t, compose, "MIGRATION_VERSION=10")
	require.NotContains(t, compose, "MIGRATION_VERSION=9")
	require.NotContains(t, compose, `"MIGRATION_VERSION"] = "9"`)
	require.NotContains(t, compose, "retained version 9")

	readme := readRepoFile(t, "README.md")
	require.Contains(t, readme, "clean version 10")
	require.Contains(t, readme, "MIGRATION_VERSION=10")
	require.Contains(t, readme, "(version 10)")
	require.NotContains(t, readme, "clean version 9")
	require.NotContains(t, readme, "MIGRATION_VERSION=9")
	require.NotContains(t, readme, "1 through 9")

	runbook := readRepoFile(t, "docs", "design", "database-migrations.md")
	require.Contains(t, runbook, "clean version 10 only")
	require.Contains(t, runbook, "CORE-IRT-004")
	require.Contains(t, runbook, "## Version 10: interview runtime")
	require.Contains(t, runbook, "000010_interview_runtime")
	require.Contains(t, runbook, "interview_progress")
	require.Contains(t, runbook, "interview_session_commands")
	require.Contains(t, runbook, "Versions 1–10 are immutable")
	require.NotContains(t, runbook, "MIGRATION_VERSION=9")
	require.NotContains(t, runbook, "version=9")
}

// TestInterviewRuntimeMigrationOnPostgres proves clean 9 to clean 10, constraints, and disposable down/up.
func TestInterviewRuntimeMigrationOnPostgres(t *testing.T) {
	db, runner := openMigrationDatabase(t)
	ctx := context.Background()

	t.Run("clean 9 migrates to clean 10 with no seed rows and keeps pins", func(t *testing.T) {
		resetSchema(t, db)
		migrateTo(t, runner, 7)
		seedVersionSevenPin(t, db)
		migrateTo(t, runner, 8)
		migrateTo(t, runner, 9)
		require.Equal(t, 1, countRows(t, db, "interview_executions"))
		migrateTo(t, runner, 10)
		require.NoError(t, schema.RequireCompatible(ctx, db))
		require.Equal(t, 1, countRows(t, db, "interview_executions"), "existing pins stay")
		for _, table := range interviewRuntimeTables {
			require.Zero(t, countRows(t, db, table), table)
		}
		var columns int
		require.NoError(t, db.QueryRowContext(ctx, `SELECT count(*) FROM information_schema.columns WHERE table_schema='public' AND table_name='interview_executions'`).Scan(&columns))
		require.Equal(t, 5, columns, "interview_executions gains no column")
		migrateTo(t, runner, 10)
	})

	t.Run("progress constraints", func(t *testing.T) {
		resetSchema(t, db)
		migrateTo(t, runner, 9)
		seedVersionSevenPin(t, db)
		migrateTo(t, runner, 10)
		insert := func(stage string, limit int) error {
			_, err := db.ExecContext(ctx, `INSERT INTO interview_progress(execution_id,follow_up_limit,current_stage,stage_activated_at,committed_version,created_at,updated_at)
VALUES ($1,$2,$3,'2026-01-02T00:00:00Z',1,'2026-01-02T00:00:00Z','2026-01-02T00:00:00Z')`, fixturePin, limit, stage)
			return err
		}
		require.NoError(t, insert("background", 2))
		require.Error(t, insert("background", 2), "one progress row per execution")
		require.ErrorContains(t, insert("waiting", 2), "violates check constraint")
		require.ErrorContains(t, insert("background", 3), "violates check constraint")
	})

	t.Run("disposable down keeps earlier tables and up recreates", func(t *testing.T) {
		resetSchema(t, db)
		migrateTo(t, runner, 9)
		seedVersionSevenPin(t, db)
		migrateTo(t, runner, 10)
		got, dirty, err := runner.Execute(ctx, schema.Options{Action: "down", Steps: 1, Disposable: true})
		require.NoError(t, err)
		require.Equal(t, 9, got)
		require.False(t, dirty)
		for _, table := range interviewRuntimeTables {
			var absent bool
			require.NoError(t, db.QueryRowContext(ctx, "SELECT to_regclass('public.'||$1) IS NULL", table).Scan(&absent))
			require.True(t, absent, table)
		}
		require.Equal(t, 1, countRows(t, db, "interview_executions"))
		migrateTo(t, runner, 10)
		require.Zero(t, countRows(t, db, "interview_progress"))
	})

	t.Run("baseline stamps the catalog it matches", func(t *testing.T) {
		resetSchema(t, db)
		migrateTo(t, runner, 10)
		_, err := db.ExecContext(ctx, "DROP TABLE schema_migrations")
		require.NoError(t, err)
		got, dirty, err := runner.Execute(ctx, schema.Options{Action: "baseline"})
		require.NoError(t, err)
		require.Equal(t, 10, got)
		require.False(t, dirty)

		_, err = db.ExecContext(ctx, "DROP TABLE schema_migrations; DROP TABLE public.interview_handoffs")
		require.NoError(t, err)
		_, _, err = runner.Execute(ctx, schema.Options{Action: "baseline"})
		require.Error(t, err, "a partial interview runtime catalog is not adopted")

		resetSchema(t, db)
		migrateTo(t, runner, 9)
		_, err = db.ExecContext(ctx, "DROP TABLE schema_migrations")
		require.NoError(t, err)
		got, _, err = runner.Execute(ctx, schema.Options{Action: "baseline"})
		require.NoError(t, err)
		require.Equal(t, 9, got, "a version 9 catalog stays version 9")
	})
}
