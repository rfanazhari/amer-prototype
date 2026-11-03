package interviewreportrepository_test

import (
	"context"
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/rfanazhari/amer-prototype/application/repository/interviewreportport"
	"github.com/stretchr/testify/assert"
)

// confirmationCase selects a SQL confirmation transaction outcome.
type confirmationCase struct {
	name      string
	wantErr   bool
	confirmed bool
	jobErr    bool
	outboxErr bool
	commitErr bool
}

// TestStoreConfirmOutbox checks atomic confirmation, idempotency and isolation from generation outcomes.
func TestStoreConfirmOutbox(t *testing.T) {
	failure := errors.New("database failure")
	tests := []confirmationCase{
		{name: "confirms and advances only pending job"},
		{name: "already confirmed is idempotent", confirmed: true},
		{name: "job update failure rolls back", wantErr: true, jobErr: true},
		{name: "outbox failure rolls back job update", wantErr: true, outboxErr: true},
		{name: "commit failure is returned", wantErr: true, commitErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store, mock := sqlStore(t)
			mock.ExpectBegin()
			rows := sqlmock.NewRows(outboxColumns)
			if tt.confirmed {
				rows.AddRow(outboxID, "interview_report", jobID, admittedAt, admittedAt, admittedAt)
			} else {
				rows.AddRow(outboxID, "interview_report", jobID, admittedAt, admittedAt, nil)
			}
			mock.ExpectQuery(`SELECT \* FROM "outbox_messages" WHERE id = \$1 AND job_kind = \$2 AND job_id = \$3`).
				WithArgs(outboxID, "interview_report", jobID, 1).WillReturnRows(rows)
			if !tt.confirmed {
				job := mock.ExpectExec(`UPDATE "interview_report_jobs" SET "state"=\$1,"updated_at"=\$2 WHERE id = \$3 AND state = \$4`).
					WithArgs("published", admittedAt, jobID, "pending")
				if tt.jobErr {
					job.WillReturnError(failure)
				} else {
					job.WillReturnResult(sqlmock.NewResult(0, 1))
					outbox := mock.ExpectExec(`UPDATE "outbox_messages" SET "confirmed_at"=\$1 WHERE id = \$2 AND confirmed_at IS NULL`).WithArgs(admittedAt, outboxID)
					if tt.outboxErr {
						outbox.WillReturnError(failure)
					} else {
						outbox.WillReturnResult(sqlmock.NewResult(0, 1))
					}
				}
			}
			if tt.jobErr || tt.outboxErr {
				mock.ExpectRollback()
			} else if tt.commitErr {
				mock.ExpectCommit().WillReturnError(failure)
			} else {
				mock.ExpectCommit()
			}
			err := store.ConfirmOutbox(context.Background(), interviewreportport.OutboxMessage{ID: outboxID, JobKind: "interview_report", JobID: jobID}, admittedAt)
			assert.Equal(t, tt.wantErr, err != nil)
			if tt.wantErr {
				assert.ErrorIs(t, err, failure)
			}
		})
	}
}
