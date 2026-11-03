package integration_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/rfanazhari/amer-prototype/application/repository/executionport"
	"github.com/rfanazhari/amer-prototype/application/service"
	"github.com/rfanazhari/amer-prototype/application/usecase/interviewreportusecase"
	"github.com/rfanazhari/amer-prototype/infrastructure/database/transaction"
	"github.com/rfanazhari/amer-prototype/infrastructure/repository/interviewreportrepository"
	"github.com/rfanazhari/amer-prototype/infrastructure/service/clock"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// reportFixture holds a stored interview execution and a real report store for admission proofs.
type reportFixture struct {
	executionFixture
	executionID string
	reports     interviewreportrepository.Store
}

// newReportFixture stores an interview execution on the release schema and removes report rows on cleanup.
func newReportFixture(t *testing.T) reportFixture {
	t.Helper()
	f := reportFixture{executionFixture: newExecutionFixture(t)}
	f.reports = interviewreportrepository.Store{DB: f.db}
	snap, fact := f.admit(t)
	f.executionID = snap.ID
	require.NoError(t, f.store.WithAssignment(context.Background(), f.assignmentID, func(ctx context.Context, _ executionport.AssignmentRow) error {
		return f.store.InsertInterview(ctx, f.assignmentID, snap, fact)
	}))
	t.Cleanup(func() { f.deleteReportRows(t) })
	return f
}

// deleteReportRows removes every report row stored for the fixture execution, children first.
func (f reportFixture) deleteReportRows(t *testing.T) {
	t.Helper()
	generations := `(SELECT g.id FROM interview_report_generations g JOIN interview_reports r ON r.id = g.report_id WHERE r.execution_id = ?)`
	jobs := `(SELECT id FROM interview_report_jobs WHERE generation_id IN ` + generations + `)`
	for _, statement := range []string{
		`DELETE FROM outbox_messages WHERE job_id IN ` + jobs,
		`DELETE FROM interview_report_attempts WHERE job_id IN ` + jobs,
		`DELETE FROM interview_report_jobs WHERE generation_id IN ` + generations,
		`DELETE FROM interview_report_statements WHERE generation_id IN ` + generations,
		`DELETE FROM interview_report_generations WHERE report_id IN (SELECT id FROM interview_reports WHERE execution_id = ?)`,
		`DELETE FROM interview_reports WHERE execution_id = ?`,
	} {
		require.NoError(t, f.db.Exec(statement, f.executionID).Error)
	}
}

// admitUseCase wires the production Admit to the real store; outboxID, when set, replaces the minted outbox identity.
func (f reportFixture) admitUseCase(outboxID string) interviewreportusecase.Admit {
	ids := []string{uuid.NewString(), uuid.NewString(), uuid.NewString(), uuid.NewString()}
	if outboxID != "" {
		ids[3] = outboxID
	}
	next := 0
	return interviewreportusecase.Admit{
		Transactions: f.reports, Completion: f.reports, Reports: f.reports, Admissions: f.reports,
		Clock: clock.System{},
		NewID: func() string { next++; return ids[next-1] },
	}
}

// interviewRows returns the assignment and execution rows as JSON text, so every column is compared.
func (f reportFixture) interviewRows(t *testing.T, db *gorm.DB) (assignment, executionRow string) {
	t.Helper()
	require.NoError(t, db.Raw(`SELECT row_to_json(a)::text FROM interview_assignments a WHERE id = ?`, f.assignmentID).Scan(&assignment).Error)
	require.NoError(t, db.Raw(`SELECT row_to_json(e)::text FROM interview_executions e WHERE id = ?`, f.executionID).Scan(&executionRow).Error)
	require.NotEmpty(t, assignment)
	require.NotEmpty(t, executionRow)
	return assignment, executionRow
}

// reportRowCount counts the report, generation, job and outbox rows stored for the fixture execution.
func (f reportFixture) reportRowCount(t *testing.T) (reports, generations, jobs, outbox int64) {
	t.Helper()
	scope := `FROM interview_reports r JOIN interview_report_generations g ON g.report_id = r.id `
	reports = f.count(t, `SELECT count(*) FROM interview_reports WHERE execution_id = ?`, f.executionID)
	generations = f.count(t, `SELECT count(*) `+scope+`WHERE r.execution_id = ?`, f.executionID)
	jobs = f.count(t, `SELECT count(*) `+scope+`JOIN interview_report_jobs j ON j.generation_id = g.id WHERE r.execution_id = ?`, f.executionID)
	outbox = f.count(t, `SELECT count(*) `+scope+`JOIN interview_report_jobs j ON j.generation_id = g.id `+
		`JOIN outbox_messages o ON o.job_id = j.id WHERE r.execution_id = ?`, f.executionID)
	return reports, generations, jobs, outbox
}

// completeInterview is the test stand-in for the Execution Lifecycle completion write.
func (f reportFixture) completeInterview(t *testing.T, tx *gorm.DB) {
	t.Helper()
	require.NoError(t, tx.Exec(`UPDATE interview_assignments SET completed_at = now() WHERE id = ?`, f.assignmentID).Error)
}

// admitInput names the fixture's assignment and execution for Admit.
func (f reportFixture) admitInput() interviewreportusecase.AdmitInput {
	return interviewreportusecase.AdmitInput{AssignmentID: f.assignmentID, ExecutionID: f.executionID}
}

// errForcedOutbox marks the failure the outbox collision is expected to surface through.
var errForcedOutbox = errors.New("forced outbox failure")

// reportCase names one real PostgreSQL report admission proof.
type reportCase struct {
	name   string
	verify func(t *testing.T, f reportFixture)
}

// TestInterviewReportAdmissionPostgres proves admission atomicity and report failure isolation on PostgreSQL.
func TestInterviewReportAdmissionPostgres(t *testing.T) {
	tests := []reportCase{
		{name: "commit keeps completion, generation, job and outbox together", verify: proveAdmissionCommit},
		{name: "forced outbox failure rolls back completion and report rows", verify: proveAdmissionRollback},
		{name: "generation failure leaves assignment and execution rows unchanged", verify: proveFailureIsolation},
		{name: "admit errors on null completion and writes no interview or report row", verify: proveAdmitNeedsCompletion},
		{name: "admit does not itself update assignment or execution", verify: proveAdmitReadOnlyToInterview},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.verify(t, newReportFixture(t))
		})
	}
}

// proveAdmissionCommit shows one transaction leaves completed_at, the pending generation, its job and one outbox row.
func proveAdmissionCommit(t *testing.T, f reportFixture) {
	ctx := context.Background()
	_, executionBefore := f.interviewRows(t, f.db)
	require.NoError(t, f.reports.InTransaction(ctx, func(ctx context.Context) error {
		f.completeInterview(t, transaction.Session(ctx, f.db))
		_, err := f.admitUseCase("").Execute(ctx, f.admitInput())
		return err
	}))
	reports, generations, jobs, outbox := f.reportRowCount(t)
	require.EqualValues(t, []int64{1, 1, 1, 1}, []int64{reports, generations, jobs, outbox})
	require.EqualValues(t, 1, f.count(t, `SELECT count(*) FROM interview_assignments WHERE id = ? AND completed_at IS NOT NULL`, f.assignmentID))
	require.EqualValues(t, 1, f.count(t, `SELECT count(*) FROM interview_report_generations g JOIN interview_reports r ON r.id = g.report_id `+
		`WHERE r.execution_id = ? AND r.assignment_id = ? AND g.generation_number = 1 AND g.state = 'pending'`, f.executionID, f.assignmentID))
	require.EqualValues(t, 1, f.count(t, `SELECT count(*) FROM outbox_messages WHERE confirmed_at IS NULL AND job_kind = 'interview_report' `+
		`AND job_id IN (SELECT j.id FROM interview_report_jobs j JOIN interview_report_generations g ON g.id = j.generation_id `+
		`JOIN interview_reports r ON r.id = g.report_id WHERE r.execution_id = ?)`, f.executionID))
	_, executionAfter := f.interviewRows(t, f.db)
	require.Equal(t, executionBefore, executionAfter)
}

// proveAdmissionRollback shows an outbox insert failure removes completed_at and every report row.
func proveAdmissionRollback(t *testing.T, f reportFixture) {
	ctx := context.Background()
	assignmentBefore, executionBefore := f.interviewRows(t, f.db)
	collidingOutbox := uuid.NewString()
	require.NoError(t, f.db.Exec(`INSERT INTO outbox_messages (id, job_kind, job_id, created_at) VALUES (?, 'interview_report', ?, now())`,
		collidingOutbox, uuid.NewString()).Error)
	t.Cleanup(func() {
		require.NoError(t, f.db.Exec(`DELETE FROM outbox_messages WHERE id = ?`, collidingOutbox).Error)
	})
	err := f.reports.InTransaction(ctx, func(ctx context.Context) error {
		f.completeInterview(t, transaction.Session(ctx, f.db))
		if _, err := f.admitUseCase(collidingOutbox).Execute(ctx, f.admitInput()); err != nil {
			return errors.Join(errForcedOutbox, err)
		}
		return nil
	})
	require.ErrorIs(t, err, errForcedOutbox)
	reports, generations, jobs, outbox := f.reportRowCount(t)
	require.EqualValues(t, []int64{0, 0, 0, 0}, []int64{reports, generations, jobs, outbox})
	assignmentAfter, executionAfter := f.interviewRows(t, f.db)
	require.Equal(t, assignmentBefore, assignmentAfter)
	require.Equal(t, executionBefore, executionAfter)
	require.EqualValues(t, 1, f.count(t, `SELECT count(*) FROM interview_assignments WHERE id = ? AND completed_at IS NULL`, f.assignmentID))
}

// proveFailureIsolation shows failing the generation later writes only report tables.
func proveFailureIsolation(t *testing.T, f reportFixture) {
	ctx := context.Background()
	require.NoError(t, f.reports.InTransaction(ctx, func(ctx context.Context) error {
		f.completeInterview(t, transaction.Session(ctx, f.db))
		_, err := f.admitUseCase("").Execute(ctx, f.admitInput())
		return err
	}))
	assignmentBefore, executionBefore := f.interviewRows(t, f.db)
	require.NoError(t, f.reports.InTransaction(ctx, func(ctx context.Context) error {
		tx := transaction.Session(ctx, f.db)
		require.NoError(t, tx.Exec(`UPDATE interview_report_generations SET state = 'failed', failure_reason = 'provider_failed', updated_at = now() `+
			`WHERE report_id IN (SELECT id FROM interview_reports WHERE execution_id = ?)`, f.executionID).Error)
		return tx.Exec(`UPDATE interview_report_jobs SET state = 'failed', updated_at = now() `+
			`WHERE generation_id IN (SELECT g.id FROM interview_report_generations g JOIN interview_reports r ON r.id = g.report_id WHERE r.execution_id = ?)`,
			f.executionID).Error
	}))
	require.EqualValues(t, 1, f.count(t, `SELECT count(*) FROM interview_report_generations g JOIN interview_reports r ON r.id = g.report_id `+
		`WHERE r.execution_id = ? AND g.state = 'failed' AND g.failure_reason = 'provider_failed'`, f.executionID))
	assignmentAfter, executionAfter := f.interviewRows(t, f.db)
	require.Equal(t, assignmentBefore, assignmentAfter)
	require.Equal(t, executionBefore, executionAfter)
}

// proveAdmitNeedsCompletion shows a null completed_at makes Admit fail and leaves no row behind.
func proveAdmitNeedsCompletion(t *testing.T, f reportFixture) {
	ctx := context.Background()
	assignmentBefore, executionBefore := f.interviewRows(t, f.db)
	err := f.reports.InTransaction(ctx, func(ctx context.Context) error {
		_, err := f.admitUseCase("").Execute(ctx, f.admitInput())
		return err
	})
	require.ErrorIs(t, err, service.ErrReportNotCompleted)
	reports, generations, jobs, outbox := f.reportRowCount(t)
	require.EqualValues(t, []int64{0, 0, 0, 0}, []int64{reports, generations, jobs, outbox})
	assignmentAfter, executionAfter := f.interviewRows(t, f.db)
	require.Equal(t, assignmentBefore, assignmentAfter)
	require.Equal(t, executionBefore, executionAfter)
}

// proveAdmitReadOnlyToInterview shows the rows Admit leaves in the open transaction equal the rows it found.
func proveAdmitReadOnlyToInterview(t *testing.T, f reportFixture) {
	ctx := context.Background()
	require.NoError(t, f.reports.InTransaction(ctx, func(ctx context.Context) error {
		tx := transaction.Session(ctx, f.db)
		f.completeInterview(t, tx)
		assignmentBefore, executionBefore := f.interviewRows(t, tx)
		if _, err := f.admitUseCase("").Execute(ctx, f.admitInput()); err != nil {
			return err
		}
		assignmentAfter, executionAfter := f.interviewRows(t, tx)
		require.Equal(t, assignmentBefore, assignmentAfter)
		require.Equal(t, executionBefore, executionAfter)
		return nil
	}))
}
