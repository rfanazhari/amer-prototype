package interviewrepository_test

import (
	"context"
	"database/sql/driver"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/rfanazhari/amer-prototype/application/repository/interviewport"
	"github.com/rfanazhari/amer-prototype/infrastructure/repository/interviewrepository"
	"github.com/stretchr/testify/assert"
)

// insertCase names one initial-progress write.
type insertCase struct {
	name    string
	record  func(t *testing.T) interviewport.ProgressRecord
	expect  func(f storeFixture)
	wantErr error
}

// expectInsertWrites expects the progress row, session shell and pending coverage with explicit zero and null columns.
func expectInsertWrites(f storeFixture) {
	f.mock.ExpectExec(`(?i)INSERT INTO "interview_progress"`).
		WithArgs(execID, 2, "background", base, int64(1), nil, "[]", 0, 0, nil, now, now).
		WillReturnResult(sqlmock.NewResult(0, 1))
	f.mock.ExpectExec(`(?i)INSERT INTO "interview_voice_sessions"`).
		WithArgs(sessionOne, execID, "background", attemptOne, nil, nil, nil, base, nil).
		WillReturnResult(sqlmock.NewResult(0, 1))
	args := make([]driver.Value, 0, 27)
	for _, q := range []string{"background", "screening", "compensation"} {
		args = append(args, sqlmock.AnyArg(), execID, q, q, "pending", nil, 0, nil, nil, nil, now)
	}
	f.mock.ExpectExec(`(?i)INSERT INTO "interview_coverage_items"`).WithArgs(args...).WillReturnResult(sqlmock.NewResult(0, 3))
}

// TestStoreInsert proves atomic creation, duplicate classification and refusal of unusable sessions.
func TestStoreInsert(t *testing.T) {
	tests := []insertCase{
		{
			name:   "writes progress version one, session shell and pending coverage in one transaction",
			record: freshRecord,
			expect: func(f storeFixture) {
				f.mock.ExpectBegin()
				expectInsertWrites(f)
				f.mock.ExpectCommit()
			},
		},
		{
			name:   "existing execution progress is a conflict and rolls back",
			record: freshRecord,
			expect: func(f storeFixture) {
				f.mock.ExpectBegin()
				f.mock.ExpectExec(`(?i)INSERT INTO "interview_progress"`).WillReturnError(&pgconn.PgError{Code: "23505"})
				f.mock.ExpectRollback()
			},
			wantErr: interviewport.ErrConflict,
		},
		{
			name: "session without an attempt id is refused before any SQL",
			record: func(t *testing.T) interviewport.ProgressRecord {
				rec := freshRecord(t)
				rec.Sessions[0].AttemptID = ""
				return rec
			},
			expect:  func(storeFixture) {},
			wantErr: interviewrepository.ErrUnmappable,
		},
		{
			name: "record naming another execution is refused before any SQL",
			record: func(t *testing.T) interviewport.ProgressRecord {
				rec := freshRecord(t)
				rec.Snapshot.ExecutionID = "other"
				return rec
			},
			expect:  func(storeFixture) {},
			wantErr: interviewrepository.ErrUnmappable,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newFixture(t)
			tt.expect(f)

			err := f.store.Insert(context.Background(), interviewport.ProgressWrite{ExecutionID: execID, AttemptID: attemptOne}, tt.record(t))

			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				return
			}
			assert.NoError(t, err)
		})
	}
}
