package interviewreportrepository_test

import (
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/rfanazhari/amer-prototype/domain/aggregate/interviewreport"
	"github.com/rfanazhari/amer-prototype/infrastructure/repository/interviewreportrepository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

const (
	reportID     = "11111111-1111-4111-8111-111111111111"
	executionID  = "22222222-2222-4222-8222-222222222222"
	assignmentID = "33333333-3333-4333-8333-333333333333"
	generationID = "44444444-4444-4444-8444-444444444444"
	secondGenID  = "44444444-4444-4444-8444-444444444445"
	jobID        = "55555555-5555-4555-8555-555555555555"
	outboxID     = "66666666-6666-4666-8666-666666666666"
	entryID      = "77777777-7777-4777-8777-777777777777"
)

// storeCase names one report persistence behavior over sqlmock.
type storeCase struct {
	name string
	run  func(t *testing.T, store interviewreportrepository.Store, mock sqlmock.Sqlmock)
}

// admittedAt is the controlled admission instant used by every case.
var admittedAt = time.Date(2026, 11, 3, 8, 0, 0, 0, time.UTC)

// reportColumns lists the interview_reports columns read by the store.
var reportColumns = []string{"id", "execution_id", "assignment_id", "created_at"}

// generationColumns lists the interview_report_generations columns read by the store.
var generationColumns = []string{"id", "report_id", "generation_number", "state", "failure_reason", "created_at", "updated_at"}

// statementColumns lists the interview_report_statements columns read by the store.
var statementColumns = []string{"id", "generation_id", "ordinal", "kind", "body", "evidence_entry_id", "evidence_revision", "presents_as_answer"}

// jobColumns lists the interview_report_jobs columns read by the claim.
var jobColumns = []string{
	"id", "generation_id", "state", "attempt_count", "fencing_token", "created_at", "wait_deadline_at",
	"window_started_at", "deadline_at", "eligible_at", "attempt_deadline_at", "updated_at",
}

// outboxColumns lists the outbox_messages columns read by the claim.
var outboxColumns = []string{"id", "job_kind", "job_id", "created_at", "publish_claimed_at", "confirmed_at"}

// sqlStore opens a GORM store over sqlmock that checks every expectation was met.
func sqlStore(t *testing.T) (interviewreportrepository.Store, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() {
		assert.NoError(t, mock.ExpectationsWereMet())
		_ = db.Close()
	})
	gdb, err := gorm.Open(postgres.New(postgres.Config{Conn: db, DriverName: "postgres"}), &gorm.Config{
		Logger:                 logger.Default.LogMode(logger.Silent),
		SkipDefaultTransaction: true,
		DisableAutomaticPing:   true,
	})
	require.NoError(t, err)
	return interviewreportrepository.Store{DB: gdb}, mock
}

// pendingReport builds a report with one pending generation through the domain factory.
func pendingReport(t *testing.T) interviewreport.Snapshot {
	t.Helper()
	report, err := interviewreport.NewReport(reportID, executionID, assignmentID, admittedAt)
	require.NoError(t, err)
	_, err = report.StartGeneration(generationID, admittedAt)
	require.NoError(t, err)
	return report.Snapshot()
}

// succeededThenPending builds a report whose first generation succeeded with all three statement kinds
// and whose second generation is pending, through the domain factory.
func succeededThenPending(t *testing.T) interviewreport.Snapshot {
	t.Helper()
	report, err := interviewreport.NewReport(reportID, executionID, assignmentID, admittedAt)
	require.NoError(t, err)
	_, err = report.StartGeneration(generationID, admittedAt)
	require.NoError(t, err)
	_, err = report.Run(generationID, admittedAt.Add(time.Second))
	require.NoError(t, err)
	var statements []interviewreport.Statement
	for _, in := range []interviewreport.StatementInput{
		{Kind: interviewreport.KindObservation, Body: "The candidate described backend work.", PresentsAsAnswer: true, Citation: &interviewreport.Citation{EntryID: entryID, Revision: 2}},
		{Kind: interviewreport.KindGap, Body: "Compensation was not answered."},
		{Kind: interviewreport.KindUncertainty, Body: "The start date was unclear."},
	} {
		statement, err := interviewreport.NewStatement(in)
		require.NoError(t, err)
		statements = append(statements, statement)
	}
	_, err = report.Succeed(generationID, statements, admittedAt.Add(2*time.Second))
	require.NoError(t, err)
	_, err = report.StartGeneration(secondGenID, admittedAt.Add(time.Minute))
	require.NoError(t, err)
	return report.Snapshot()
}
