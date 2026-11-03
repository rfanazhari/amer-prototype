package schema_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"regexp"
	"strings"
	"testing"

	"github.com/rfanazhari/amer-prototype/infrastructure/database/schema"
	"github.com/stretchr/testify/require"
)

// interviewReportingTables are the seven tables version 12 adds, in creation order.
var interviewReportingTables = []string{
	"interview_reports",
	"interview_report_generations",
	"interview_report_statements",
	"interview_report_commands",
	"interview_report_jobs",
	"interview_report_attempts",
	"outbox_messages",
}

// interviewReportingUp and interviewReportingDown are the version 12 file names.
const (
	interviewReportingUp   = "000012_interview_reporting.up.sql"
	interviewReportingDown = "000012_interview_reporting.down.sql"
)

// TestInterviewReportingMigrationArtifacts proves the version 12 SQL shape, scope and drop order.
func TestInterviewReportingMigrationArtifacts(t *testing.T) {
	up := readRepoFile(t, "migrations", "pgsql", interviewReportingUp)
	down := readRepoFile(t, "migrations", "pgsql", interviewReportingDown)

	for _, table := range interviewReportingTables {
		require.Contains(t, up, "CREATE TABLE public."+table+" (", table)
	}
	require.Equal(t, 7, strings.Count(up, "CREATE TABLE"), "only the seven reporting tables are created")

	for _, snippet := range []string{
		"execution_id uuid NOT NULL",
		"assignment_id uuid NOT NULL",
		"REFERENCES public.interview_executions(id) ON UPDATE RESTRICT ON DELETE RESTRICT",
		"REFERENCES public.interview_assignments(id) ON UPDATE RESTRICT ON DELETE RESTRICT",
		"REFERENCES public.administrative_accounts(id) ON UPDATE RESTRICT ON DELETE RESTRICT",
		"generation_number > 0",
		"UNIQUE (report_id, generation_number)",
		"'pending', 'running', 'succeeded', 'failed'",
		"updated_at >= created_at",
		"'observation', 'gap', 'uncertainty'",
		"char_length(body) BETWEEN 1 AND 2000",
		"evidence_revision > 0",
		"UNIQUE (generation_id, ordinal)",
		"presents_as_answer boolean NOT NULL",
		"regenerate-interview-report/v1",
		"intent_fingerprint ~ '^[0-9a-f]{64}$'",
		"UNIQUE (scope, command_kind, request_key)",
		"'pending', 'published', 'waiting', 'claimed', 'succeeded', 'failed', 'expired'",
		"attempt_count >= 0 AND attempt_count <= 2",
		"fencing_token > 0",
		"wait_deadline_at = created_at + interval '30 minutes'",
		"deadline_at <= window_started_at + interval '5 minutes'",
		"attempt_number IN (1, 2)",
		"UNIQUE (job_id, attempt_number)",
		"deadline_at <= started_at + interval '60 seconds'",
		"'succeeded', 'failed', 'expired', 'superseded'",
		"job_kind = 'interview_report'",
		"interview_report_generations_one_active_INDEX",
		"WHERE state IN ('pending', 'running')",
		"outbox_messages_unconfirmed_INDEX",
		"WHERE confirmed_at IS NULL",
		"interview_report_jobs_eligible_INDEX",
		"WHERE state IN ('pending', 'published')",
	} {
		require.Contains(t, up, snippet, snippet)
	}

	require.NotContains(t, up, "INSERT INTO", "no seed rows")
	for _, forbidden := range []string{"AutoMigrate", "ADD COLUMN", "DROP COLUMN", "ON DELETE CASCADE", "deleted_at", "recommendation", "rank", "salary"} {
		require.NotContains(t, up, forbidden)
	}
	require.NotRegexp(t, regexp.MustCompile(`(?i)ALTER TABLE`), up, "existing tables gain nothing")

	// dependencyCase names a child table that must disappear before its referenced parent.
	type dependencyCase struct {
		name   string
		child  string
		parent string
	}
	dependencies := []dependencyCase{
		{name: "statements before generations", child: "interview_report_statements", parent: "interview_report_generations"},
		{name: "commands before generations", child: "interview_report_commands", parent: "interview_report_generations"},
		{name: "attempts before jobs", child: "interview_report_attempts", parent: "interview_report_jobs"},
		{name: "jobs before generations", child: "interview_report_jobs", parent: "interview_report_generations"},
		{name: "generations before reports", child: "interview_report_generations", parent: "interview_reports"},
	}
	require.Equal(t, 7, strings.Count(down, "DROP TABLE"), "rollback owns only the seven reporting tables")
	require.NotContains(t, down, "CASCADE", "rollback must not remove outside dependents")
	require.Contains(t, down, "DROP TABLE IF EXISTS public.outbox_messages;")
	for _, tt := range dependencies {
		t.Run(tt.name, func(t *testing.T) {
			child := strings.Index(down, "DROP TABLE IF EXISTS public."+tt.child+";")
			parent := strings.Index(down, "DROP TABLE IF EXISTS public."+tt.parent+";")
			require.GreaterOrEqual(t, child, 0)
			require.GreaterOrEqual(t, parent, 0)
			require.Less(t, child, parent, "child must drop before its referenced parent")
		})
	}
	for _, kept := range []string{"interview_executions", "interview_assignments", "administrative_accounts", "interview_progress"} {
		require.NotRegexp(t, regexp.MustCompile(`DROP TABLE[^;]*public\.`+kept+`\b`), down, kept+" is not dropped")
	}
}

// TestInterviewReportingChecksums proves the new files are pinned and versions 1 through 11 are untouched.
func TestInterviewReportingChecksums(t *testing.T) {
	var sums map[string]string
	require.NoError(t, json.Unmarshal([]byte(readRepoFile(t, "infrastructure", "database", "schema", "checksums.json")), &sums))
	manifest := readRepoFile(t, "docs", "design", "database-migration-checksums.sha256")
	for _, name := range []string{interviewReportingUp, interviewReportingDown} {
		body := []byte(readRepoFile(t, "migrations", "pgsql", name))
		digest := sha256.Sum256(body)
		want := hex.EncodeToString(digest[:])
		require.Equal(t, want, sums[name], name)
		require.Contains(t, manifest, want+"  migrations/pgsql/"+name)
	}
	require.Len(t, sums, 24, "versions 1 through 12, up and down")
	require.Equal(t, "bf1bde8379e4f705be46a238bee5744e746380195a164b37fd73421d9e55cdce", sums["000010_interview_runtime.up.sql"])
	require.Equal(t, "8af2b71e7103d14326e475b885dc86c4c2ec558e124c3c20846c45e73f81702f", sums["000010_interview_runtime.down.sql"])
	require.Equal(t, "7a4c38363aec441b9c755abd456eef7a1e709f44d57da4f5317058326224c9f5", sums["000011_interview_runtime_integration.up.sql"])
	require.Equal(t, "5aaecaad8c6c3106261adaf9ac0ed51ac203aed18b93e879d7741516fe7390d1", sums["000011_interview_runtime_integration.down.sql"])
}

// TestReportingReleaseGrantsAndRunbook proves the schema floor and runtime grants include reporting.
func TestReportingReleaseGrantsAndRunbook(t *testing.T) {
	require.Equal(t, 12, schema.ReleaseVersion)
	require.Subset(t, schema.RuntimeGrantTables, interviewReportingTables)
	require.Subset(t, schema.RuntimeGrantTables, interviewRuntimeTables, "version 10 grants stay")

	runbook := readRepoFile(t, "docs", "design", "database-migrations.md")
	require.Contains(t, runbook, "clean version 12 only")
	require.Contains(t, runbook, "000012_interview_reporting")
	require.Contains(t, runbook, "Versions 1–11 are immutable")
}

// TestInterviewReportingMigrationOnPostgres proves clean 11 to clean 12, constraints, and disposable down/up.
func TestInterviewReportingMigrationOnPostgres(t *testing.T) {
	db, runner := openMigrationDatabase(t)
	ctx := context.Background()

	resetSchema(t, db)
	migrateTo(t, runner, 7)
	seedVersionSevenPin(t, db)
	migrateTo(t, runner, 11)
	for _, table := range interviewReportingTables {
		var present bool
		require.NoError(t, db.QueryRowContext(ctx, `SELECT to_regclass('public.`+table+`') IS NOT NULL`).Scan(&present))
		require.False(t, present, table+" is absent at version 11")
	}
	migrateTo(t, runner, 12)
	require.NoError(t, schema.RequireCompatible(ctx, db))
	for _, table := range interviewReportingTables {
		require.Equal(t, 0, countRows(t, db, table), table+" starts empty")
	}
	require.Equal(t, 1, countRows(t, db, "interview_executions"), "existing pins stay")

	const (
		reportID      = "20000000-0000-4000-8000-000000000001"
		generationOne = "20000000-0000-4000-8000-000000000002"
		generationTwo = "20000000-0000-4000-8000-000000000003"
		jobOne        = "20000000-0000-4000-8000-000000000004"
		evidenceID    = "20000000-0000-4000-8000-000000000005"
	)
	exec := func(t *testing.T, statement string, args ...any) error {
		t.Helper()
		_, err := db.ExecContext(ctx, statement, args...)
		return err
	}
	require.NoError(t, exec(t, `INSERT INTO interview_reports(id,execution_id,assignment_id,created_at) VALUES ($1,$2,$3,now())`, reportID, fixturePin, fixtureAssignment))
	require.NoError(t, exec(t, `INSERT INTO interview_report_generations(id,report_id,generation_number,state,created_at,updated_at) VALUES ($1,$2,1,'pending',now(),now())`, generationOne, reportID))

	tests := []struct {
		name      string
		statement string
		args      []any
	}{
		{name: "second report for the execution", statement: `INSERT INTO interview_reports(id,execution_id,assignment_id,created_at) VALUES ('20000000-0000-4000-8000-0000000000a1',$1,$2,now())`, args: []any{fixturePin, fixtureAssignment}},
		{name: "second active generation", statement: `INSERT INTO interview_report_generations(id,report_id,generation_number,state,created_at,updated_at) VALUES ($1,$2,2,'running',now(),now())`, args: []any{generationTwo, reportID}},
		{name: "duplicate generation number", statement: `INSERT INTO interview_report_generations(id,report_id,generation_number,state,failure_reason,created_at,updated_at) VALUES ($1,$2,1,'failed','x',now(),now())`, args: []any{generationTwo, reportID}},
		{name: "failed generation without reason", statement: `INSERT INTO interview_report_generations(id,report_id,generation_number,state,created_at,updated_at) VALUES ($1,$2,2,'failed',now(),now())`, args: []any{generationTwo, reportID}},
		{name: "answer without evidence", statement: `INSERT INTO interview_report_statements(id,generation_id,ordinal,kind,body,presents_as_answer) VALUES ('20000000-0000-4000-8000-0000000000b1',$1,0,'observation','candidate answered',true)`, args: []any{generationOne}},
		{name: "gap with evidence", statement: `INSERT INTO interview_report_statements(id,generation_id,ordinal,kind,body,evidence_entry_id,evidence_revision,presents_as_answer) VALUES ('20000000-0000-4000-8000-0000000000b2',$1,0,'gap','not covered',$2,1,false)`, args: []any{generationOne, evidenceID}},
		{name: "unknown statement kind", statement: `INSERT INTO interview_report_statements(id,generation_id,ordinal,kind,body,presents_as_answer) VALUES ('20000000-0000-4000-8000-0000000000b3',$1,0,'recommendation','hire',false)`, args: []any{generationOne}},
		{name: "empty statement body", statement: `INSERT INTO interview_report_statements(id,generation_id,ordinal,kind,body,presents_as_answer) VALUES ('20000000-0000-4000-8000-0000000000b4',$1,0,'uncertainty','',false)`, args: []any{generationOne}},
		{name: "job attempt count above budget", statement: `INSERT INTO interview_report_jobs(id,generation_id,state,attempt_count,fencing_token,created_at,wait_deadline_at,eligible_at,updated_at) VALUES ($1,$2,'pending',3,1,now(),now()+interval '30 minutes',now(),now())`, args: []any{jobOne, generationOne}},
		{name: "outbox kind outside this version", statement: `INSERT INTO outbox_messages(id,job_kind,job_id,created_at) VALUES ('20000000-0000-4000-8000-0000000000c1','document_execution',$1,now())`, args: []any{jobOne}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Error(t, exec(t, tt.statement, tt.args...))
		})
	}

	require.NoError(t, exec(t, `INSERT INTO interview_report_statements(id,generation_id,ordinal,kind,body,presents_as_answer) VALUES ('20000000-0000-4000-8000-0000000000b5',$1,0,'gap','not covered',false)`, generationOne))
	require.NoError(t, exec(t, `UPDATE interview_report_generations SET state='failed', failure_reason='provider_unavailable', updated_at=now() WHERE id=$1`, generationOne))
	require.NoError(t, exec(t, `INSERT INTO interview_report_generations(id,report_id,generation_number,state,created_at,updated_at) VALUES ($1,$2,2,'pending',now(),now())`, generationTwo, reportID), "a new generation is allowed after failure")

	version, dirty, err := runner.Execute(ctx, schema.Options{Action: "down", Steps: 1, Disposable: true})
	require.NoError(t, err)
	require.Equal(t, 11, version)
	require.False(t, dirty)
	for _, table := range interviewReportingTables {
		var present bool
		require.NoError(t, db.QueryRowContext(ctx, `SELECT to_regclass('public.`+table+`') IS NOT NULL`).Scan(&present))
		require.False(t, present, table+" is dropped by down")
	}
	require.Equal(t, 1, countRows(t, db, "interview_executions"), "down keeps the execution pin")
	require.Equal(t, 1, countRows(t, db, "interview_assignments"), "down keeps the assignment")
	migrateTo(t, runner, 12)
	require.NoError(t, schema.RequireCompatible(ctx, db))

	t.Run("baseline recognizes reporting tables", func(t *testing.T) {
		require.NoError(t, exec(t, `DROP TABLE schema_migrations`))
		version, _, err := runner.Execute(ctx, schema.Options{Action: "baseline"})
		require.NoError(t, err)
		require.Equal(t, 12, version)
	})
}
