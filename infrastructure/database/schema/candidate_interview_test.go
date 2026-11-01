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

// candidateInterviewTables are the two tables version 9 adds, in dependency order.
var candidateInterviewTables = []string{"interview_consents", "interview_controllers"}

// candidateInterviewUp and candidateInterviewDown are the version 9 file names.
const (
	candidateInterviewUp   = "000009_candidate_interview.up.sql"
	candidateInterviewDown = "000009_candidate_interview.down.sql"
)

// TestCandidateInterviewMigrationArtifacts proves the version 9 SQL shape, scope and drop order.
func TestCandidateInterviewMigrationArtifacts(t *testing.T) {
	up := readRepoFile(t, "migrations", "pgsql", candidateInterviewUp)
	down := readRepoFile(t, "migrations", "pgsql", candidateInterviewDown)

	for _, table := range candidateInterviewTables {
		require.Contains(t, up, "CREATE TABLE public."+table+" (", table)
	}
	require.Equal(t, 2, strings.Count(up, "CREATE TABLE"), "only the two interview tables are created")

	for _, snippet := range []string{
		// interview_consents
		"assignment_id uuid NOT NULL",
		"notice_version text NOT NULL",
		"accepted_at timestamptz NOT NULL",
		"candidate_session_id uuid NOT NULL",
		"notice_version ~ '^[A-Za-z0-9][A-Za-z0-9._:-]{0,63}$'",
		"UNIQUE (assignment_id, notice_version)",
		"REFERENCES public.interview_assignments(id) ON UPDATE RESTRICT ON DELETE RESTRICT",
		"REFERENCES public.candidate_sessions(id) ON UPDATE RESTRICT ON DELETE RESTRICT",
		"ON public.interview_consents USING btree (assignment_id)",
		// interview_controllers
		"assignment_id uuid PRIMARY KEY",
		"execution_id uuid NOT NULL",
		"controller_id text NOT NULL",
		"version bigint NOT NULL",
		"updated_at timestamptz NOT NULL",
		"controller_id ~ '^[A-Za-z0-9][A-Za-z0-9._:-]{0,63}$'",
		"version >= 1",
		"REFERENCES public.interview_executions(id) ON UPDATE RESTRICT ON DELETE RESTRICT",
	} {
		require.Contains(t, up, snippet)
	}
	require.Equal(t, 2, strings.Count(up, "REFERENCES public.interview_assignments(id)"), "both tables reference the assignment")

	require.NotContains(t, up, "INSERT INTO", "no notice seed and no seeder rows")
	for _, forbidden := range []string{"AutoMigrate", "ADD COLUMN", "DROP COLUMN", "ON DELETE CASCADE", "deleted_at", "withdrawn_at"} {
		require.NotContains(t, up, forbidden)
	}
	require.NotRegexp(t, regexp.MustCompile(`(?i)notice_body|statement|report|transcript|salary`), up, "no notice text and no report storage")
	require.NotRegexp(t, regexp.MustCompile(`(?i)ALTER TABLE[^;]*interview_executions`), up, "the execution pin table gains nothing")

	consents := strings.Index(down, "DROP TABLE IF EXISTS public.interview_consents")
	controllers := strings.Index(down, "DROP TABLE IF EXISTS public.interview_controllers")
	require.GreaterOrEqual(t, controllers, 0)
	require.Greater(t, consents, controllers, "controllers drop before consents")
	for _, kept := range []string{"interview_assignments", "candidate_sessions", "interview_executions"} {
		require.NotRegexp(t, regexp.MustCompile(`DROP TABLE[^;]*public\.`+kept+`\b`), down, kept+" is not dropped")
	}
}

// TestCandidateInterviewChecksums proves the new files are pinned and versions 1 through 8 are untouched.
func TestCandidateInterviewChecksums(t *testing.T) {
	var sums map[string]string
	require.NoError(t, json.Unmarshal([]byte(readRepoFile(t, "infrastructure", "database", "schema", "checksums.json")), &sums))
	manifest := readRepoFile(t, "docs", "design", "database-migration-checksums.sha256")
	for _, name := range []string{candidateInterviewUp, candidateInterviewDown} {
		body := []byte(readRepoFile(t, "migrations", "pgsql", name))
		digest := sha256.Sum256(body)
		want := hex.EncodeToString(digest[:])
		require.Equal(t, want, sums[name], name)
		require.Contains(t, manifest, want+"  migrations/pgsql/"+name)
	}
	require.Len(t, sums, 18, "versions 1 through 9, up and down")
	require.Equal(t, "8208d541363390aafdf37ef21138b58626f5357756b0b4fb1684f1d7b3cb0740", sums["000008_execution_lifecycle.up.sql"])
	require.Equal(t, "f213ae6700e1424e42897ca8c02d08833a62fc59468b11ed697a3c8d223941c4", sums["000008_execution_lifecycle.down.sql"])
	require.Equal(t, "fad47701239f7dd8ecf4d060904a19697ae177dcea39a0508ba85f96825b4b78", sums["000007_interview_executions.up.sql"])
	require.Equal(t, "55a758f53320f47737a270a30a43d0ac9ddf20d760f886e177cbdd9c72304909", sums["000007_interview_executions.down.sql"])
}

// TestReleaseVersionRequiresNine proves the clean schema floor, grants, and operator messages.
func TestReleaseVersionRequiresNine(t *testing.T) {
	require.Equal(t, 9, schema.ReleaseVersion)
	require.Subset(t, schema.RuntimeGrantTables, candidateInterviewTables)
	require.Subset(t, schema.RuntimeGrantTables, lifecycleTables, "version 8 grants stay")

	tests := []struct {
		name    string
		rows    *sqlmock.Rows
		err     error
		want    string
		wantErr bool
	}{
		{name: "unreadable", err: errors.New("private"), want: "clean version 9", wantErr: true},
		{name: "floor eight is refused", rows: sqlmock.NewRows([]string{"version", "dirty"}).AddRow(8, false), want: "require clean version 9", wantErr: true},
		{name: "future is refused", rows: sqlmock.NewRows([]string{"version", "dirty"}).AddRow(10, false), want: "require clean version 9", wantErr: true},
		{name: "dirty is refused", rows: sqlmock.NewRows([]string{"version", "dirty"}).AddRow(9, true), want: "require clean version 9", wantErr: true},
		{name: "ambiguous", rows: sqlmock.NewRows([]string{"version", "dirty"}).AddRow(9, false).AddRow(9, false), want: "exactly one clean version 9", wantErr: true},
		{name: "clean nine", rows: sqlmock.NewRows([]string{"version", "dirty"}).AddRow(9, false)},
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
	_, _, err := runner.Execute(context.Background(), schema.Options{Action: "up", Version: 10})
	require.ErrorContains(t, err, "1 through 9")
	_, _, err = runner.Execute(context.Background(), schema.Options{Action: "up", Version: 9})
	require.EqualError(t, err, "database required")
	_, _, err = runner.Execute(context.Background(), schema.Options{Action: "down", Steps: 9, Disposable: true})
	require.EqualError(t, err, "database required")
}

// TestOperationalFilesRequireVersionNine proves operator-facing files move to the same floor.
func TestOperationalFilesRequireVersionNine(t *testing.T) {
	makefile := readRepoFile(t, "Makefile")
	require.Contains(t, makefile, "version=9 dirty=false")
	require.Contains(t, makefile, "make docker.migrate MIGRATION_VERSION=9")
	require.NotContains(t, makefile, "version=8 dirty=false")
	require.NotContains(t, makefile, "MIGRATION_VERSION=8")

	compose := readRepoFile(t, "scripts", "verify_compose.py")
	require.Contains(t, compose, "MIGRATION_VERSION=9")
	require.NotContains(t, compose, "MIGRATION_VERSION=8")
	require.NotContains(t, compose, `"MIGRATION_VERSION"] = "8"`)
	require.NotContains(t, compose, "retained version 8")

	readme := readRepoFile(t, "README.md")
	require.Contains(t, readme, "clean version 9")
	require.Contains(t, readme, "MIGRATION_VERSION=9")
	require.Contains(t, readme, "(version 9)")
	require.NotContains(t, readme, "clean version 8")
	require.NotContains(t, readme, "MIGRATION_VERSION=8")
	require.NotContains(t, readme, "1 through 8")

	runbook := readRepoFile(t, "docs", "design", "database-migrations.md")
	require.Contains(t, runbook, "clean version 9 only")
	require.Contains(t, runbook, "CORE-CIN-004")
	require.Contains(t, runbook, "## Version 9: candidate interview")
	require.Contains(t, runbook, "000009_candidate_interview")
	require.Contains(t, runbook, "interview_consents")
	require.Contains(t, runbook, "interview_controllers")
	require.Contains(t, runbook, "Versions 1–9 are immutable")
	require.NotContains(t, runbook, "MIGRATION_VERSION=8")
	require.NotContains(t, runbook, "version=8")
}

// TestCandidateInterviewMigrationOnPostgres proves clean 8 to clean 9, constraints, and disposable down/up.
func TestCandidateInterviewMigrationOnPostgres(t *testing.T) {
	db, runner := openMigrationDatabase(t)
	ctx := context.Background()

	const (
		session   = "00000000-0000-0000-0000-0000000000a9"
		consent   = "00000000-0000-0000-0000-0000000000c9"
		secondRow = "00000000-0000-0000-0000-0000000000c8"
	)
	seedSession := func(t *testing.T) {
		t.Helper()
		seedVersionSevenPin(t, db)
		_, err := db.ExecContext(ctx, `INSERT INTO interview_invitations(id,assignment_id,token_hash,created_by,created_at,expires_at) VALUES
('00000000-0000-0000-0000-0000000000a8',$1,repeat('b',64),$2,'2026-01-02Z','2026-01-09Z');
INSERT INTO candidate_sessions(id,assignment_id,invitation_id,token_hash,created_at,expires_at) VALUES
($3,$1,'00000000-0000-0000-0000-0000000000a8',repeat('a',64),'2026-01-02Z','2026-01-02T02:00:00Z')`, fixtureAssignment, fixtureAccount, session)
		require.NoError(t, err)
	}
	insertConsent := func(id, version string) error {
		_, err := db.ExecContext(ctx, `INSERT INTO interview_consents(id,assignment_id,notice_version,accepted_at,candidate_session_id)
VALUES ($1,$2,$3,'2026-01-02T00:05:00Z',$4)`, id, fixtureAssignment, version, session)
		return err
	}

	t.Run("clean 8 migrates to clean 9 with no seed rows", func(t *testing.T) {
		resetSchema(t, db)
		migrateTo(t, runner, 8)
		migrateTo(t, runner, 9)
		require.NoError(t, schema.RequireCompatible(ctx, db))
		for _, table := range candidateInterviewTables {
			require.Zero(t, countRows(t, db, table), table)
		}
		var columns int
		require.NoError(t, db.QueryRowContext(ctx, `SELECT count(*) FROM information_schema.columns WHERE table_schema='public' AND table_name='interview_executions'`).Scan(&columns))
		require.Equal(t, 5, columns, "interview_executions gains no column")
		migrateTo(t, runner, 9)
	})

	t.Run("consent constraints", func(t *testing.T) {
		resetSchema(t, db)
		migrateTo(t, runner, 8)
		seedSession(t)
		migrateTo(t, runner, 9)
		require.NoError(t, insertConsent(consent, "notice-261102"))
		require.ErrorContains(t, insertConsent(secondRow, "notice-261102"), "interview_consents_assignment_id_notice_version_key", "one acceptance per assignment and version")
		require.NoError(t, insertConsent(secondRow, "notice-261103"), "a later version is kept beside the first")
		for _, bad := range []string{"", " notice", "-notice", strings.Repeat("a", 65)} {
			require.ErrorContains(t, insertConsent("00000000-0000-0000-0000-0000000000c7", bad), "violates check constraint", bad)
		}
		_, err := db.ExecContext(ctx, `DELETE FROM public.candidate_sessions WHERE id=$1`, session)
		require.ErrorContains(t, err, "violates foreign key constraint", "the provenance session cannot vanish")
		_, err = db.ExecContext(ctx, `DELETE FROM public.interview_assignments WHERE id=$1`, fixtureAssignment)
		require.Error(t, err, "restrict, no cascade")
	})

	t.Run("controller constraints", func(t *testing.T) {
		resetSchema(t, db)
		migrateTo(t, runner, 8)
		seedSession(t)
		migrateTo(t, runner, 9)
		insert := func(assignment, execution, controller string, version int) error {
			_, err := db.ExecContext(ctx, `INSERT INTO interview_controllers(assignment_id,execution_id,controller_id,version,updated_at)
VALUES ($1,$2,$3,$4,'2026-01-02T00:06:00Z')`, assignment, execution, controller, version)
			return err
		}
		require.NoError(t, insert(fixtureAssignment, fixturePin, "tab-1", 1))
		require.Error(t, insert(fixtureAssignment, fixturePin, "tab-2", 2), "one controller row per assignment")
		require.ErrorContains(t, insert("00000000-0000-0000-0000-0000000000f2", fixturePin, "tab-1", 1), "violates foreign key constraint")
		_, err := db.ExecContext(ctx, `UPDATE interview_controllers SET version=0`)
		require.ErrorContains(t, err, "violates check constraint")
		_, err = db.ExecContext(ctx, `UPDATE interview_controllers SET controller_id=' tab'`)
		require.ErrorContains(t, err, "violates check constraint")
		_, err = db.ExecContext(ctx, `DELETE FROM public.interview_executions WHERE id=$1`, fixturePin)
		require.ErrorContains(t, err, "violates foreign key constraint", "the pinned execution cannot vanish")
	})

	t.Run("disposable down keeps earlier tables and up recreates", func(t *testing.T) {
		resetSchema(t, db)
		migrateTo(t, runner, 8)
		seedSession(t)
		migrateTo(t, runner, 9)
		got, dirty, err := runner.Execute(ctx, schema.Options{Action: "down", Steps: 1, Disposable: true})
		require.NoError(t, err)
		require.Equal(t, 8, got)
		require.False(t, dirty)
		for _, table := range candidateInterviewTables {
			var absent bool
			require.NoError(t, db.QueryRowContext(ctx, "SELECT to_regclass('public.'||$1) IS NULL", table).Scan(&absent))
			require.True(t, absent, table)
		}
		for _, table := range []string{"interview_assignments", "candidate_sessions", "interview_executions", "agent_executions"} {
			require.Equal(t, 1, countRows(t, db, table), table)
		}
		migrateTo(t, runner, 9)
		require.Zero(t, countRows(t, db, "interview_consents"))
	})

	t.Run("baseline stamps the catalog it matches", func(t *testing.T) {
		resetSchema(t, db)
		migrateTo(t, runner, 9)
		_, err := db.ExecContext(ctx, "DROP TABLE schema_migrations")
		require.NoError(t, err)
		got, dirty, err := runner.Execute(ctx, schema.Options{Action: "baseline"})
		require.NoError(t, err)
		require.Equal(t, 9, got)
		require.False(t, dirty)

		_, err = db.ExecContext(ctx, "DROP TABLE schema_migrations; DROP TABLE public.interview_controllers")
		require.NoError(t, err)
		_, _, err = runner.Execute(ctx, schema.Options{Action: "baseline"})
		require.Error(t, err, "a partial interview catalog is not adopted")

		resetSchema(t, db)
		migrateTo(t, runner, 8)
		_, err = db.ExecContext(ctx, "DROP TABLE schema_migrations")
		require.NoError(t, err)
		got, _, err = runner.Execute(ctx, schema.Options{Action: "baseline"})
		require.NoError(t, err)
		require.Equal(t, 8, got, "a version 8 catalog stays version 8")
	})
}
