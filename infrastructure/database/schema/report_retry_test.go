package schema_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"testing"

	"github.com/rfanazhari/amer-prototype/infrastructure/database/schema"
	"github.com/stretchr/testify/require"
)

// TestReportRetryMigrationArtifacts verifies the correction is a new, checksum-pinned constraint-only migration.
func TestReportRetryMigrationArtifacts(t *testing.T) {
	// testCase names one checksum-pinned correction artifact.
	type testCase struct{ name string }
	tests := []testCase{{name: "000013_interview_report_retry_window.up.sql"}, {name: "000013_interview_report_retry_window.down.sql"}}
	var sums map[string]string
	require.NoError(t, json.Unmarshal([]byte(readRepoFile(t, "infrastructure", "database", "schema", "checksums.json")), &sums))
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body := readRepoFile(t, "migrations", "pgsql", tt.name)
			digest := sha256.Sum256([]byte(body))
			require.Equal(t, hex.EncodeToString(digest[:]), sums[tt.name])
			require.Contains(t, body, "ALTER TABLE public.interview_report_jobs")
			require.Contains(t, body, "interview_report_jobs_unwindowed_states_check")
			require.NotContains(t, body, "UPDATE public.")
			require.NotContains(t, body, "DELETE FROM")
			require.NotContains(t, body, "CREATE TABLE")
		})
	}
}

// TestReportRetryMigrationPostgres proves populated 12-to-13 retry-window compatibility and safe disposable reversal.
func TestReportRetryMigrationPostgres(t *testing.T) {
	db, runner := openMigrationDatabase(t)
	ctx := context.Background()
	resetSchema(t, db)
	migrateTo(t, runner, 7)
	seedVersionSevenPin(t, db)
	migrateTo(t, runner, 12)
	const reportID = "20000000-0000-4000-8000-000000000011"
	const generationID = "20000000-0000-4000-8000-000000000012"
	const jobID = "20000000-0000-4000-8000-000000000013"
	_, err := db.ExecContext(ctx, `INSERT INTO interview_reports(id,execution_id,assignment_id,created_at) VALUES ($1,$2,$3,now())`, reportID, fixturePin, fixtureAssignment)
	require.NoError(t, err)
	_, err = db.ExecContext(ctx, `INSERT INTO interview_report_generations(id,report_id,generation_number,state,created_at,updated_at) VALUES ($1,$2,1,'running',now(),now())`, generationID, reportID)
	require.NoError(t, err)
	_, err = db.ExecContext(ctx, `INSERT INTO interview_report_jobs(id,generation_id,state,attempt_count,fencing_token,created_at,wait_deadline_at,window_started_at,deadline_at,eligible_at,updated_at) VALUES ($1,$2,'claimed',1,2,now(),now()+interval '30 minutes',now(),now()+interval '5 minutes',now(),now())`, jobID, generationID)
	require.NoError(t, err)
	var before string
	require.NoError(t, db.QueryRowContext(ctx, `SELECT row_to_json(j)::text FROM interview_report_jobs j WHERE id=$1`, jobID).Scan(&before))
	_, err = db.ExecContext(ctx, `UPDATE interview_report_jobs SET state='pending' WHERE id=$1`, jobID)
	require.Error(t, err, "version 12 refuses the required retry")
	migrateTo(t, runner, 13)
	require.NoError(t, schema.RequireCompatible(ctx, db))
	var after string
	require.NoError(t, db.QueryRowContext(ctx, `SELECT row_to_json(j)::text FROM interview_report_jobs j WHERE id=$1`, jobID).Scan(&after))
	require.Equal(t, before, after, "migration does not rewrite jobs")
	// testCase describes one independent public behavior scenario.
	type testCase struct {
		name, state string
		count       int
		wantErr     bool
	}
	tests := []testCase{{name: "pending retry", state: "pending", count: 1}, {name: "published retry", state: "published", count: 1}, {name: "waiting cannot retain window", state: "waiting", count: 1, wantErr: true}, {name: "unattempted cannot own window", state: "pending", count: 0, wantErr: true}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := db.ExecContext(ctx, `UPDATE interview_report_jobs SET state=$1,attempt_count=$2 WHERE id=$3`, tt.state, tt.count, jobID)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
	_, _, err = runner.Execute(ctx, schema.Options{Action: "down", Steps: 1, Disposable: true})
	require.Error(t, err, "down refuses retained retry windows instead of rewriting them")
	// Failed transactional DDL preserves the corrected constraint and every job field.
	require.NoError(t, db.QueryRowContext(ctx, `SELECT state FROM interview_report_jobs WHERE id=$1`, jobID).Scan(&after))
	require.Equal(t, "published", after)
	// Reset only this explicitly disposable schema to exercise a clean down/up.
	resetSchema(t, db)
	migrateTo(t, runner, 13)
	version, dirty, err := runner.Execute(ctx, schema.Options{Action: "down", Steps: 1, Disposable: true})
	require.NoError(t, err)
	require.Equal(t, 12, version)
	require.False(t, dirty)
	migrateTo(t, runner, 13)
	require.NoError(t, schema.RequireCompatible(ctx, db))
}
