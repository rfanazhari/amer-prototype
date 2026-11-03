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

// admission builds the admission of the pending report with the controlled wait deadline.
func admission(t *testing.T) interviewreportport.Admission {
	t.Helper()
	return interviewreportport.Admission{
		Report:         pendingReport(t),
		JobID:          jobID,
		OutboxID:       outboxID,
		WaitDeadlineAt: admittedAt.Add(interviewreport.EvidenceWait),
	}
}

// expectAdmissionInserts expects the report, generation, job and outbox rows in that order.
func expectAdmissionInserts(mock sqlmock.Sqlmock) {
	mock.ExpectExec(`INSERT INTO "interview_reports"`).
		WithArgs(reportID, executionID, assignmentID, admittedAt).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(`INSERT INTO "interview_report_generations"`).
		WithArgs(generationID, reportID, 1, "pending", nil, admittedAt, admittedAt).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(`INSERT INTO "interview_report_jobs"`).
		WithArgs(jobID, generationID, "pending", 0, int64(1), admittedAt, admittedAt.Add(30*time.Minute), nil, nil, admittedAt, nil, admittedAt).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(`INSERT INTO "outbox_messages"`).
		WithArgs(outboxID, "interview_report", jobID, admittedAt, nil, nil).
		WillReturnResult(sqlmock.NewResult(1, 1))
}

// TestStoreInsert covers the admission rows, their atomicity and the tables it must not touch.
func TestStoreInsert(t *testing.T) {
	failure := errors.New("outbox unavailable")
	tests := []storeCase{
		{
			name: "writes report, first generation, job and outbox in one transaction",
			run: func(t *testing.T, store interviewreportrepository.Store, mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				expectAdmissionInserts(mock)
				mock.ExpectCommit()
				require.NoError(t, store.Insert(context.Background(), admission(t)))
			},
		},
		{
			name: "joins the caller transaction and leaves the commit to the caller",
			run: func(t *testing.T, store interviewreportrepository.Store, mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				expectAdmissionInserts(mock)
				mock.ExpectCommit()
				err := store.InTransaction(context.Background(), func(ctx context.Context) error {
					return store.Insert(ctx, admission(t))
				})
				require.NoError(t, err)
			},
		},
		{
			name: "rolls back every row when the outbox insert fails",
			run: func(t *testing.T, store interviewreportrepository.Store, mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectExec(`INSERT INTO "interview_reports"`).WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectExec(`INSERT INTO "interview_report_generations"`).WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectExec(`INSERT INTO "interview_report_jobs"`).WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectExec(`INSERT INTO "outbox_messages"`).WillReturnError(failure)
				mock.ExpectRollback()
				err := store.Insert(context.Background(), admission(t))
				assert.ErrorIs(t, err, failure)
			},
		},
		{
			name: "refuses an admission whose report does not hold exactly one pending generation",
			run: func(t *testing.T, store interviewreportrepository.Store, mock sqlmock.Sqlmock) {
				bad := admission(t)
				bad.Report = succeededThenPending(t)
				err := store.Insert(context.Background(), bad)
				assert.ErrorIs(t, err, interviewreport.ErrInvalidReport)
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

// jobRow returns a claimable job row; window is the stored window start, or the zero time for none.
func jobRow(attempts int, token int64, window time.Time) *sqlmock.Rows {
	var started, deadline any
	if !window.IsZero() {
		started, deadline = window, window.Add(interviewreport.JobWindow)
	}
	return sqlmock.NewRows(jobColumns).AddRow(
		jobID, generationID, "published", attempts, token, admittedAt, admittedAt.Add(interviewreport.EvidenceWait),
		started, deadline, admittedAt, nil, admittedAt,
	)
}

// TestStoreClaimJob covers the locked job claim and the attempt it inserts.
func TestStoreClaimJob(t *testing.T) {
	const claimQuery = `FROM interview_report_jobs WHERE state IN \('pending', 'published'\) AND eligible_at <= \$1 AND attempt_count < \$2 AND \(deadline_at IS NULL OR deadline_at > \$3\) ORDER BY eligible_at, id LIMIT 1 FOR UPDATE SKIP LOCKED`
	claimedAt := admittedAt.Add(time.Minute)
	tests := []storeCase{
		{
			name: "first claim starts the job window and a 60 second attempt",
			run: func(t *testing.T, store interviewreportrepository.Store, mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectQuery(claimQuery).WithArgs(claimedAt, interviewreport.MaxAttempts, claimedAt).WillReturnRows(jobRow(0, 1, time.Time{}))
				mock.ExpectExec(`INSERT INTO "interview_report_attempts"`).
					WithArgs(sqlmock.AnyArg(), jobID, 1, int64(2), claimedAt, claimedAt.Add(60*time.Second), nil, nil).
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectExec(`UPDATE "interview_report_jobs" SET "attempt_count"=\$1,"attempt_deadline_at"=\$2,"deadline_at"=\$3,"fencing_token"=\$4,"state"=\$5,"updated_at"=\$6,"window_started_at"=\$7 WHERE id = \$8`).
					WithArgs(1, claimedAt.Add(60*time.Second), claimedAt.Add(5*time.Minute), int64(2), "claimed", claimedAt, claimedAt, jobID).
					WillReturnResult(sqlmock.NewResult(0, 1))
				mock.ExpectCommit()
				got, err := store.ClaimJob(context.Background(), claimedAt)
				require.NoError(t, err)
				assert.Equal(t, interviewreportport.ClaimedJob{
					JobID: jobID, GenerationID: generationID, AttemptNumber: 1, FencingToken: 2,
					StartedAt: claimedAt, AttemptDeadlineAt: claimedAt.Add(60 * time.Second), JobDeadlineAt: claimedAt.Add(5 * time.Minute),
				}, got)
			},
		},
		{
			name: "second claim keeps the window and caps the attempt at the job deadline",
			run: func(t *testing.T, store interviewreportrepository.Store, mock sqlmock.Sqlmock) {
				window := admittedAt
				second := window.Add(5*time.Minute - 20*time.Second)
				mock.ExpectBegin()
				mock.ExpectQuery(claimQuery).WithArgs(second, interviewreport.MaxAttempts, second).WillReturnRows(jobRow(1, 2, window))
				mock.ExpectExec(`INSERT INTO "interview_report_attempts"`).
					WithArgs(sqlmock.AnyArg(), jobID, 2, int64(3), second, window.Add(5*time.Minute), nil, nil).
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectExec(`UPDATE "interview_report_jobs" SET`).
					WithArgs(2, window.Add(5*time.Minute), window.Add(5*time.Minute), int64(3), "claimed", second, window, jobID).
					WillReturnResult(sqlmock.NewResult(0, 1))
				mock.ExpectCommit()
				got, err := store.ClaimJob(context.Background(), second)
				require.NoError(t, err)
				assert.Equal(t, 2, got.AttemptNumber)
				assert.Equal(t, int64(3), got.FencingToken)
				assert.Equal(t, window.Add(5*time.Minute), got.AttemptDeadlineAt)
			},
		},
		{
			name: "reports no eligible job and rolls back",
			run: func(t *testing.T, store interviewreportrepository.Store, mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectQuery(claimQuery).WithArgs(claimedAt, interviewreport.MaxAttempts, claimedAt).
					WillReturnRows(sqlmock.NewRows(jobColumns))
				mock.ExpectRollback()
				_, err := store.ClaimJob(context.Background(), claimedAt)
				assert.ErrorIs(t, err, interviewreportport.ErrNothingToClaim)
			},
		},
		{
			name: "does not report a claim when the attempt insert fails",
			run: func(t *testing.T, store interviewreportrepository.Store, mock sqlmock.Sqlmock) {
				failure := errors.New("duplicate attempt")
				mock.ExpectBegin()
				mock.ExpectQuery(claimQuery).WithArgs(claimedAt, interviewreport.MaxAttempts, claimedAt).WillReturnRows(jobRow(0, 1, time.Time{}))
				mock.ExpectExec(`INSERT INTO "interview_report_attempts"`).WillReturnError(failure)
				mock.ExpectRollback()
				got, err := store.ClaimJob(context.Background(), claimedAt)
				assert.ErrorIs(t, err, failure)
				assert.Equal(t, interviewreportport.ClaimedJob{}, got)
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

// TestStoreClaimOutbox covers the locked outbox batch claim.
func TestStoreClaimOutbox(t *testing.T) {
	const claimQuery = `FROM outbox_messages WHERE confirmed_at IS NULL ORDER BY created_at, id LIMIT \$1 FOR UPDATE SKIP LOCKED`
	claimedAt := admittedAt.Add(time.Minute)
	other := "66666666-6666-4666-8666-666666666667"
	tests := []storeCase{
		{
			name: "claims unconfirmed rows and stamps the claim instant",
			run: func(t *testing.T, store interviewreportrepository.Store, mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectQuery(claimQuery).WithArgs(10).WillReturnRows(sqlmock.NewRows(outboxColumns).
					AddRow(outboxID, "interview_report", jobID, admittedAt, nil, nil).
					AddRow(other, "interview_report", jobID, admittedAt.Add(time.Second), nil, nil))
				mock.ExpectExec(`UPDATE "outbox_messages" SET "publish_claimed_at"=\$1 WHERE id IN \(\$2,\$3\)`).
					WithArgs(claimedAt, outboxID, other).WillReturnResult(sqlmock.NewResult(0, 2))
				mock.ExpectCommit()
				got, err := store.ClaimOutbox(context.Background(), 10, claimedAt)
				require.NoError(t, err)
				assert.Equal(t, []interviewreportport.OutboxMessage{
					{ID: outboxID, JobKind: "interview_report", JobID: jobID, CreatedAt: admittedAt},
					{ID: other, JobKind: "interview_report", JobID: jobID, CreatedAt: admittedAt.Add(time.Second)},
				}, got)
			},
		},
		{
			name: "returns an empty batch without an update",
			run: func(t *testing.T, store interviewreportrepository.Store, mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectQuery(claimQuery).WithArgs(10).WillReturnRows(sqlmock.NewRows(outboxColumns))
				mock.ExpectCommit()
				got, err := store.ClaimOutbox(context.Background(), 10, claimedAt)
				require.NoError(t, err)
				assert.Empty(t, got)
			},
		},
		{
			name: "rejects a non-positive batch size before touching the database",
			run: func(t *testing.T, store interviewreportrepository.Store, mock sqlmock.Sqlmock) {
				_, err := store.ClaimOutbox(context.Background(), 0, claimedAt)
				assert.Error(t, err)
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
