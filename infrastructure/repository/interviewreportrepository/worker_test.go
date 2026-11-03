package interviewreportrepository_test

import (
	"context"
	"errors"
	"github.com/DATA-DOG/go-sqlmock"
	"github.com/rfanazhari/amer-prototype/application/repository/interviewreportport"
	"github.com/rfanazhari/amer-prototype/domain/aggregate/interviewreport"
	"github.com/rfanazhari/amer-prototype/infrastructure/repository/interviewreportrepository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

// expectWorkerLock requires the generation lock before the job lock, protecting outcome and fencing.
func expectWorkerLock(m sqlmock.Sqlmock, token int64, state string) {
	m.ExpectBegin()
	m.ExpectQuery(`SELECT .* FROM "interview_report_generations".*FOR UPDATE`).WithArgs(jobID, 1).WillReturnRows(sqlmock.NewRows(generationColumns).AddRow(generationID, reportID, 1, "running", nil, admittedAt, admittedAt))
	m.ExpectQuery(`SELECT .* FROM "interview_report_jobs".*FOR UPDATE`).WithArgs(jobID, 1).WillReturnRows(sqlmock.NewRows(jobColumns).AddRow(jobID, generationID, state, 1, token, admittedAt, admittedAt.Add(30*time.Minute), admittedAt, admittedAt.Add(5*time.Minute), admittedAt, admittedAt.Add(time.Minute), admittedAt))
}

// expectWorkerReport loads a report through its public restoration shape before terminal mutation.
func expectWorkerReport(m sqlmock.Sqlmock) {
	m.ExpectQuery(`SELECT .* FROM "interview_reports"`).WithArgs(reportID, 1).WillReturnRows(sqlmock.NewRows(reportColumns).AddRow(reportID, executionID, assignmentID, admittedAt))
	m.ExpectQuery(`SELECT .* FROM "interview_report_generations"`).WithArgs(reportID).WillReturnRows(sqlmock.NewRows(generationColumns).AddRow(generationID, reportID, 1, "running", nil, admittedAt, admittedAt))
	m.ExpectQuery(`SELECT .* FROM "interview_report_statements"`).WithArgs(generationID).WillReturnRows(sqlmock.NewRows(statementColumns))
}

// TestStoreFinishAttempt proves report-only atomic writes, stale fences, failed output and rollback error propagation with sqlmock.
func TestStoreFinishAttempt(t *testing.T) {
	failure := errors.New("write failed")
	// testCase describes one independent public behavior scenario.
	type testCase struct {
		name      string
		wantErr   bool
		token     int64
		reason    string
		failWrite bool
	}
	tests := []testCase{
		{name: "accepted statements and outcome commit", token: 2},
		{name: "rejected output stores no statements and schedules retry", token: 2, reason: "output_rejected"},
		{name: "stale fencing token performs no writes", token: 3},
		{name: "statement failure rolls back outcome", token: 2, failWrite: true, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store, m := sqlStore(t)
			expectWorkerLock(m, tt.token, "claimed")
			result := interviewreportport.AttemptResult{Claim: interviewreportport.ClaimedJob{JobID: jobID, GenerationID: generationID, AttemptNumber: 1, FencingToken: 2}, FinishedAt: admittedAt.Add(time.Second), FailureReason: tt.reason}
			if tt.reason == "" {
				st, err := interviewreport.NewStatement(interviewreport.StatementInput{Kind: interviewreport.KindGap, Body: "Experience unclear."})
				require.NoError(t, err)
				result.Statements = []interviewreport.Statement{st}
			}
			if tt.token != 2 {
				m.ExpectCommit()
			} else {
				if tt.reason == "" {
					expectWorkerReport(m)
				}
				m.ExpectExec(`UPDATE "interview_report_attempts"`).WithArgs(result.FinishedAt, mockAttemptOutcome(tt.reason), jobID, 1, int64(2)).WillReturnResult(sqlmock.NewResult(0, 1))
				m.ExpectExec(`UPDATE "interview_report_jobs" SET`).WillReturnResult(sqlmock.NewResult(0, 1))
				if tt.reason == "" {
					if tt.failWrite {
						m.ExpectExec(`INSERT INTO "interview_report_statements"`).WillReturnError(failure)
						m.ExpectRollback()
					} else {
						m.ExpectExec(`INSERT INTO "interview_report_statements"`).WithArgs(sqlmock.AnyArg(), generationID, 0, "gap", "Experience unclear.", nil, nil, false).WillReturnResult(sqlmock.NewResult(1, 1))
						m.ExpectExec(`UPDATE "interview_report_generations" SET`).WithArgs(nil, "succeeded", result.FinishedAt, generationID).WillReturnResult(sqlmock.NewResult(0, 1))
						m.ExpectCommit()
					}
				} else {
					m.ExpectCommit()
				}
			}
			err := store.FinishAttempt(context.Background(), result)
			if tt.wantErr {
				assert.ErrorIs(t, err, failure)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// mockAttemptOutcome maps this test's failure input independently to its expected stored outcome.
func mockAttemptOutcome(reason string) string {
	if reason != "" {
		return "failed"
	}
	return "succeeded"
}

// TestStoreWorkerWaiting covers persistence of Q4 without an attempt insert or an interview write.
func TestStoreWorkerWaiting(t *testing.T) {
	tests := []storeCase{{name: "waiting state and null window persist", run: func(t *testing.T, s interviewreportrepository.Store, m sqlmock.Sqlmock) {
		m.ExpectBegin()
		m.ExpectQuery(`SELECT .* FROM "interview_report_generations".*FOR UPDATE`).WithArgs(jobID, 1).WillReturnRows(sqlmock.NewRows(generationColumns).AddRow(generationID, reportID, 1, "pending", nil, admittedAt, admittedAt))
		m.ExpectQuery(`FROM interview_report_jobs WHERE id = \$1 FOR UPDATE SKIP LOCKED`).WithArgs(jobID).WillReturnRows(jobRow(0, 1, time.Time{}))
		m.ExpectExec(`UPDATE "interview_report_jobs" SET`).WithArgs(0, nil, nil, admittedAt.Add(30*time.Second), int64(1), "waiting", admittedAt, nil, jobID).WillReturnResult(sqlmock.NewResult(0, 1))
		m.ExpectCommit()
		_, err := s.ClaimAttempt(context.Background(), jobID, false, admittedAt)
		assert.ErrorIs(t, err, interviewreportport.ErrNothingToClaim)
	}}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) { s, m := sqlStore(t); tt.run(t, s, m) })
	}
}

// TestStoreTerminalWorkerDuplicate proves timezone normalization cannot reopen or rewrite an expired generation.
func TestStoreTerminalWorkerDuplicate(t *testing.T) {
	tests := []storeCase{{name: "expired duplicate is a read-only no-op", run: func(t *testing.T, s interviewreportrepository.Store, m sqlmock.Sqlmock) {
		local := admittedAt.In(time.FixedZone("database", 7*3600))
		m.ExpectBegin()
		m.ExpectQuery(`SELECT .* FROM "interview_report_generations".*FOR UPDATE`).WithArgs(jobID, 1).WillReturnRows(sqlmock.NewRows(generationColumns).AddRow(generationID, reportID, 1, "failed", "evidence_unavailable", local, local.Add(30*time.Minute)))
		m.ExpectQuery(`FROM interview_report_jobs WHERE id = \$1 FOR UPDATE SKIP LOCKED`).WithArgs(jobID).WillReturnRows(sqlmock.NewRows(jobColumns).AddRow(jobID, generationID, "expired", 0, int64(1), local, local.Add(30*time.Minute), nil, nil, local, nil, local.Add(30*time.Minute)))
		m.ExpectCommit()
		_, err := s.ClaimAttempt(context.Background(), jobID, true, admittedAt.Add(31*time.Minute))
		assert.ErrorIs(t, err, interviewreportport.ErrNothingToClaim)
	}}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) { s, m := sqlStore(t); tt.run(t, s, m) })
	}
}
