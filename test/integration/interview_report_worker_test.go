package integration_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/rfanazhari/amer-prototype/application/repository/interviewreportport"
	"github.com/rfanazhari/amer-prototype/domain/aggregate/interviewreport"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// admittedWorkerJob admits a completed fixture and reads its database-generated budget identity.
func admittedWorkerJob(t *testing.T, f reportFixture) (string, time.Time) {
	t.Helper()
	ctx := context.Background()
	require.NoError(t, f.reports.InTransaction(ctx, func(ctx context.Context) error {
		f.completeInterview(t, f.db)
		_, err := f.admitUseCase("").Execute(ctx, f.admitInput())
		return err
	}))
	var job interviewreportport.JobRef
	require.NoError(t, f.db.Table("interview_report_jobs AS j").Select("j.id AS job_id").Joins("JOIN interview_report_generations g ON g.id=j.generation_id").Joins("JOIN interview_reports r ON r.id=g.report_id").Where("r.execution_id = ?", f.executionID).Take(&job).Error)
	var created time.Time
	require.NoError(t, f.db.Table("interview_report_jobs").Select("created_at").Where("id = ?", job.JobID).Scan(&created).Error)
	return job.JobID, created.UTC().Truncate(time.Millisecond).Add(time.Second)
}

// TestInterviewReportWorkerPostgres proves real retry constraints, locking, atomic rollback and interview isolation.
func TestInterviewReportWorkerPostgres(t *testing.T) {
	tests := []reportCase{
		{name: "retry preserves original deadline then atomically stores account", verify: proveWorkerRetry},
		{name: "concurrent duplicate claims create one attempt", verify: proveWorkerConcurrentClaim},
		{name: "statement failure rolls back fenced outcome", verify: proveWorkerRollback},
		{name: "waiting expiry spends no attempt", verify: proveWorkerEvidenceExpiry},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) { tt.verify(t, newReportFixture(t)) })
	}
}

// proveWorkerRetry runs two attempts with a real pending/published retry window and refuses the old holder.
func proveWorkerRetry(t *testing.T, f reportFixture) {
	ctx := context.Background()
	id, now := admittedWorkerJob(t, f)
	assignmentBefore, executionBefore := f.interviewRows(t, f.db)
	ref, err := f.reports.FindJob(ctx, id)
	require.NoError(t, err)
	assert.Equal(t, f.executionID, ref.ExecutionID)
	first, err := f.reports.ClaimAttempt(ctx, id, true, now)
	require.NoError(t, err)
	require.NoError(t, f.reports.FinishAttempt(ctx, interviewreportport.AttemptResult{Claim: first, FinishedAt: now.Add(time.Second), FailureReason: "output_rejected"}))
	_, err = f.reports.ClaimAttempt(ctx, id, true, now.Add(6*time.Second-time.Millisecond))
	require.ErrorIs(t, err, interviewreportport.ErrNothingToClaim)
	require.NoError(t, f.reports.ReconcileJob(ctx, id, true, now.Add(6*time.Second)))
	require.NoError(t, f.reports.ReconcileJob(ctx, id, true, now.Add(6*time.Second)))
	assert.EqualValues(t, 2, f.count(t, `SELECT count(*) FROM outbox_messages WHERE job_id=?`, id), "one admission and one retry intent")
	claim, err := f.reports.ClaimAttempt(ctx, id, true, now.Add(4*time.Minute+50*time.Second))
	require.NoError(t, err)
	assert.Equal(t, 2, claim.AttemptNumber)
	assert.Equal(t, now.Add(5*time.Minute), claim.AttemptDeadlineAt)
	st, err := interviewreport.NewStatement(interviewreport.StatementInput{Kind: interviewreport.KindObservation, Body: "Described backend work.", PresentsAsAnswer: true, Citation: &interviewreport.Citation{EntryID: uuid.NewString(), Revision: 1}})
	require.NoError(t, err)
	require.NoError(t, f.reports.FinishAttempt(ctx, interviewreportport.AttemptResult{Claim: first, Statements: []interviewreport.Statement{st}, FinishedAt: now.Add(4*time.Minute + 51*time.Second)}))
	assert.Zero(t, f.count(t, `SELECT count(*) FROM interview_report_statements WHERE generation_id=?`, claim.GenerationID))
	require.NoError(t, f.reports.FinishAttempt(ctx, interviewreportport.AttemptResult{Claim: claim, Statements: []interviewreport.Statement{st}, FinishedAt: now.Add(4*time.Minute + 51*time.Second)}))
	snapshot, err := f.reports.FindByExecution(ctx, f.executionID)
	require.NoError(t, err)
	assert.Equal(t, interviewreport.StateSucceeded, snapshot.Generations[0].State)
	assert.Len(t, snapshot.Generations[0].Statements, 1)
	_, err = f.reports.ClaimAttempt(ctx, id, true, now.Add(4*time.Minute+52*time.Second))
	require.ErrorIs(t, err, interviewreportport.ErrNothingToClaim)
	assignmentAfter, executionAfter := f.interviewRows(t, f.db)
	assert.Equal(t, assignmentBefore, assignmentAfter)
	assert.Equal(t, executionBefore, executionAfter)
}

// proveWorkerConcurrentClaim checks a duplicate delivery cannot create a second active attempt on PostgreSQL.
func proveWorkerConcurrentClaim(t *testing.T, f reportFixture) {
	id, now := admittedWorkerJob(t, f)
	var group sync.WaitGroup
	results := make(chan error, 2)
	for range 2 {
		group.Go(func() { _, err := f.reports.ClaimAttempt(context.Background(), id, true, now); results <- err })
	}
	group.Wait()
	close(results)
	successes := 0
	for err := range results {
		if err == nil {
			successes++
		} else {
			require.ErrorIs(t, err, interviewreportport.ErrNothingToClaim)
		}
	}
	assert.Equal(t, 1, successes)
	assert.EqualValues(t, 1, f.count(t, `SELECT count(*) FROM interview_report_attempts WHERE job_id=?`, id))
}

// proveWorkerRollback forces an invalid persisted UUID after the budget update and proves full rollback.
func proveWorkerRollback(t *testing.T, f reportFixture) {
	ctx := context.Background()
	id, now := admittedWorkerJob(t, f)
	claim, err := f.reports.ClaimAttempt(ctx, id, true, now)
	require.NoError(t, err)
	st, err := interviewreport.NewStatement(interviewreport.StatementInput{Kind: interviewreport.KindObservation, Body: "Described work.", PresentsAsAnswer: true, Citation: &interviewreport.Citation{EntryID: "not-a-uuid", Revision: 1}})
	require.NoError(t, err)
	err = f.reports.FinishAttempt(ctx, interviewreportport.AttemptResult{Claim: claim, Statements: []interviewreport.Statement{st}, FinishedAt: now.Add(time.Second)})
	require.Error(t, err)
	assert.EqualValues(t, 1, f.count(t, `SELECT count(*) FROM interview_report_jobs WHERE id=? AND state='claimed' AND fencing_token=?`, id, claim.FencingToken))
	assert.EqualValues(t, 1, f.count(t, `SELECT count(*) FROM interview_report_attempts WHERE job_id=? AND finished_at IS NULL`, id))
	assert.Zero(t, f.count(t, `SELECT count(*) FROM interview_report_statements WHERE generation_id=?`, claim.GenerationID))
}

// proveWorkerEvidenceExpiry proves waiting at admission and exact Q4 give-up through the actual repository.
func proveWorkerEvidenceExpiry(t *testing.T, f reportFixture) {
	ctx := context.Background()
	id, now := admittedWorkerJob(t, f)
	_, err := f.reports.ClaimAttempt(ctx, id, false, now)
	require.ErrorIs(t, err, interviewreportport.ErrNothingToClaim)
	assert.EqualValues(t, 1, f.count(t, `SELECT count(*) FROM interview_report_jobs WHERE id=? AND state='waiting' AND attempt_count=0 AND deadline_at IS NULL`, id))
	due, err := f.reports.DueJobs(ctx, now.Add(30*time.Minute), 20)
	require.NoError(t, err)
	require.NotEmpty(t, due)
	require.NoError(t, f.reports.ReconcileJob(ctx, id, false, now.Add(30*time.Minute-time.Second)))
	snapshot, err := f.reports.FindByExecution(ctx, f.executionID)
	require.NoError(t, err)
	assert.Equal(t, "evidence_unavailable", snapshot.Generations[0].FailureReason)
	assert.Zero(t, f.count(t, `SELECT count(*) FROM interview_report_attempts WHERE job_id=?`, id))
}
