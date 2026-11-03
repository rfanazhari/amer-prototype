package interviewreportrepository_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/rfanazhari/amer-prototype/application/repository/interviewreportport"
	"github.com/rfanazhari/amer-prototype/domain/aggregate/interviewreport"
	"github.com/rfanazhari/amer-prototype/infrastructure/repository/interviewreportrepository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestStoreInTransaction covers commit, rollback and joining an open transaction.
func TestStoreInTransaction(t *testing.T) {
	boom := errors.New("boom")
	tests := []storeCase{
		{
			name: "commits when the callback succeeds",
			run: func(t *testing.T, store interviewreportrepository.Store, mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectCommit()
				called := false
				require.NoError(t, store.InTransaction(context.Background(), func(context.Context) error {
					called = true
					return nil
				}))
				assert.True(t, called)
			},
		},
		{
			name: "rolls back and returns the callback error",
			run: func(t *testing.T, store interviewreportrepository.Store, mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectRollback()
				err := store.InTransaction(context.Background(), func(context.Context) error { return boom })
				assert.ErrorIs(t, err, boom)
			},
		},
		{
			name: "joins the transaction already carried by the context",
			run: func(t *testing.T, store interviewreportrepository.Store, mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectCommit()
				inner := false
				require.NoError(t, store.InTransaction(context.Background(), func(ctx context.Context) error {
					return store.InTransaction(ctx, func(context.Context) error {
						inner = true
						return nil
					})
				}))
				assert.True(t, inner)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store, mock := sqlStore(t)
			tt.run(t, store, mock)
		})
	}
}

// TestStoreCompletedAt covers the sibling completion marker read.
func TestStoreCompletedAt(t *testing.T) {
	const query = `SELECT .*completed_at.* FROM interview_assignments JOIN interview_executions`
	zone := time.FixedZone("WIB", 7*3600)
	tests := []storeCase{
		{
			name: "returns the completion instant in UTC",
			run: func(t *testing.T, store interviewreportrepository.Store, mock sqlmock.Sqlmock) {
				mock.ExpectQuery(query).WithArgs(assignmentID, executionID).
					WillReturnRows(sqlmock.NewRows([]string{"completed_at"}).AddRow(admittedAt.In(zone)))
				got, err := store.CompletedAt(context.Background(), assignmentID, executionID)
				require.NoError(t, err)
				require.NotNil(t, got)
				assert.Equal(t, admittedAt, *got)
				assert.Equal(t, time.UTC, got.Location())
			},
		},
		{
			name: "returns nil while the interview is not completed",
			run: func(t *testing.T, store interviewreportrepository.Store, mock sqlmock.Sqlmock) {
				mock.ExpectQuery(query).WithArgs(assignmentID, executionID).
					WillReturnRows(sqlmock.NewRows([]string{"completed_at"}).AddRow(nil))
				got, err := store.CompletedAt(context.Background(), assignmentID, executionID)
				require.NoError(t, err)
				assert.Nil(t, got)
			},
		},
		{
			name: "reports an unknown pairing as not found",
			run: func(t *testing.T, store interviewreportrepository.Store, mock sqlmock.Sqlmock) {
				mock.ExpectQuery(query).WithArgs(assignmentID, executionID).
					WillReturnRows(sqlmock.NewRows([]string{"completed_at"}))
				got, err := store.CompletedAt(context.Background(), assignmentID, executionID)
				assert.ErrorIs(t, err, interviewreportport.ErrNotFound)
				assert.Nil(t, got)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store, mock := sqlStore(t)
			tt.run(t, store, mock)
		})
	}
}

// TestStoreFindByExecution covers restoration of a report, its generations and statements in three queries.
func TestStoreFindByExecution(t *testing.T) {
	failedAt := admittedAt.Add(time.Minute)
	tests := []storeCase{
		{
			name: "restores generations in number order with their statements",
			run: func(t *testing.T, store interviewreportrepository.Store, mock sqlmock.Sqlmock) {
				want := succeededThenPending(t)
				mock.ExpectQuery(`FROM "interview_reports" WHERE execution_id = \$1`).WithArgs(executionID, 1).
					WillReturnRows(sqlmock.NewRows(reportColumns).AddRow(reportID, executionID, assignmentID, admittedAt))
				mock.ExpectQuery(`FROM "interview_report_generations" WHERE report_id = \$1 ORDER BY generation_number`).WithArgs(reportID).
					WillReturnRows(sqlmock.NewRows(generationColumns).
						AddRow(generationID, reportID, 1, "succeeded", nil, admittedAt, admittedAt.Add(2*time.Second)).
						AddRow(secondGenID, reportID, 2, "pending", nil, admittedAt.Add(time.Minute), admittedAt.Add(time.Minute)))
				mock.ExpectQuery(`FROM "interview_report_statements" WHERE generation_id IN \(\$1,\$2\) ORDER BY generation_id, ordinal`).
					WithArgs(generationID, secondGenID).
					WillReturnRows(sqlmock.NewRows(statementColumns).
						AddRow("a1", generationID, 0, "observation", "The candidate described backend work.", entryID, 2, true).
						AddRow("a2", generationID, 1, "gap", "Compensation was not answered.", nil, nil, false).
						AddRow("a3", generationID, 2, "uncertainty", "The start date was unclear.", nil, nil, false))
				got, err := store.FindByExecution(context.Background(), executionID)
				require.NoError(t, err)
				assert.Equal(t, want, got)
			},
		},
		{
			name: "restores a failed generation with its reason",
			run: func(t *testing.T, store interviewreportrepository.Store, mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`FROM "interview_reports" WHERE execution_id = \$1`).WithArgs(executionID, 1).
					WillReturnRows(sqlmock.NewRows(reportColumns).AddRow(reportID, executionID, assignmentID, admittedAt))
				mock.ExpectQuery(`FROM "interview_report_generations"`).WithArgs(reportID).
					WillReturnRows(sqlmock.NewRows(generationColumns).
						AddRow(generationID, reportID, 1, "failed", "evidence_unavailable", admittedAt, failedAt))
				mock.ExpectQuery(`FROM "interview_report_statements"`).WithArgs(generationID).
					WillReturnRows(sqlmock.NewRows(statementColumns))
				got, err := store.FindByExecution(context.Background(), executionID)
				require.NoError(t, err)
				require.Len(t, got.Generations, 1)
				assert.Equal(t, interviewreport.StateFailed, got.Generations[0].State)
				assert.Equal(t, "evidence_unavailable", got.Generations[0].FailureReason)
				assert.Equal(t, failedAt, got.Generations[0].UpdatedAt)
			},
		},
		{
			name: "reports a missing report as not found",
			run: func(t *testing.T, store interviewreportrepository.Store, mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`FROM "interview_reports"`).WithArgs(executionID, 1).
					WillReturnRows(sqlmock.NewRows(reportColumns))
				_, err := store.FindByExecution(context.Background(), executionID)
				assert.ErrorIs(t, err, interviewreportport.ErrNotFound)
			},
		},
		{
			name: "rejects a stored generation that breaks the report invariants",
			run: func(t *testing.T, store interviewreportrepository.Store, mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`FROM "interview_reports"`).WithArgs(executionID, 1).
					WillReturnRows(sqlmock.NewRows(reportColumns).AddRow(reportID, executionID, assignmentID, admittedAt))
				mock.ExpectQuery(`FROM "interview_report_generations"`).WithArgs(reportID).
					WillReturnRows(sqlmock.NewRows(generationColumns).
						AddRow(generationID, reportID, 1, "failed", nil, admittedAt, failedAt))
				mock.ExpectQuery(`FROM "interview_report_statements"`).WithArgs(generationID).
					WillReturnRows(sqlmock.NewRows(statementColumns))
				_, err := store.FindByExecution(context.Background(), executionID)
				assert.ErrorIs(t, err, interviewreport.ErrInvalidReport)
			},
		},
		{
			name: "rejects a stored answer statement without evidence",
			run: func(t *testing.T, store interviewreportrepository.Store, mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`FROM "interview_reports"`).WithArgs(executionID, 1).
					WillReturnRows(sqlmock.NewRows(reportColumns).AddRow(reportID, executionID, assignmentID, admittedAt))
				mock.ExpectQuery(`FROM "interview_report_generations"`).WithArgs(reportID).
					WillReturnRows(sqlmock.NewRows(generationColumns).
						AddRow(generationID, reportID, 1, "succeeded", nil, admittedAt, failedAt))
				mock.ExpectQuery(`FROM "interview_report_statements"`).WithArgs(generationID).
					WillReturnRows(sqlmock.NewRows(statementColumns).
						AddRow("a1", generationID, 0, "observation", "Answered.", nil, nil, true))
				_, err := store.FindByExecution(context.Background(), executionID)
				assert.ErrorIs(t, err, interviewreport.ErrCitationRequired)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store, mock := sqlStore(t)
			tt.run(t, store, mock)
		})
	}
}
