package interviewrepository_test

import (
	"context"
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/rfanazhari/amer-prototype/application/repository/interviewport"
	"github.com/rfanazhari/amer-prototype/infrastructure/database/transaction"
	"github.com/rfanazhari/amer-prototype/infrastructure/repository/interviewrepository"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// TestSessionCommandSQL checks original replay fields, nullable legacy reads and shared rollback through sqlmock.
func TestSessionCommandSQL(t *testing.T) {
	// testCase describes one SQL boundary assertion.
	type testCase struct {
		name string
		run  func(*testing.T, storeFixture, interviewrepository.Commands)
	}
	tests := []testCase{
		{name: "legacy row preserves missing answer", run: func(t *testing.T, f storeFixture, s interviewrepository.Commands) {
			f.mock.ExpectQuery(`SELECT .* FROM "interview_session_commands"`).WithArgs(execID, "key", 1).WillReturnRows(sqlmock.NewRows([]string{"id", "execution_id", "idempotency_key", "intent", "attempt_id", "state", "created_at"}).AddRow(entryOne, execID, "key", "activate", attemptOne, "legacy", base))
			got, err := s.Find(context.Background(), execID, "key")
			require.NoError(t, err)
			require.Equal(t, "legacy", got.State)
			require.Empty(t, got.AnswerSDP)
			require.Zero(t, got.ExecutionVersion)
		}},
		{name: "finalization checks ownership and keeps original answer", run: func(t *testing.T, f storeFixture, s interviewrepository.Commands) {
			f.mock.ExpectExec(`UPDATE "interview_session_commands" SET`).WithArgs("original-sdp", int64(2), "succeeded", now, execID, "key", attemptOne, sessionOne, "pending").WillReturnResult(sqlmock.NewResult(0, 1))
			require.NoError(t, s.Finalize(context.Background(), interviewport.SessionCommand{ExecutionID: execID, Key: "key", AttemptID: attemptOne, SessionID: sessionOne, AnswerSDP: "original-sdp", ExecutionVersion: 2}))
		}},
		{name: "joined result failure rolls back without nested transaction", run: func(t *testing.T, f storeFixture, s interviewrepository.Commands) {
			failure := errors.New("result write failed")
			f.mock.ExpectBegin()
			f.mock.ExpectExec(`UPDATE "interview_session_commands" SET`).WillReturnError(failure)
			f.mock.ExpectRollback()
			err := transaction.Within(context.Background(), f.store.DB, func(ctx context.Context, _ *gorm.DB) error {
				return s.Finalize(ctx, interviewport.SessionCommand{ExecutionID: execID, Key: "key", AttemptID: attemptOne, SessionID: sessionOne, AnswerSDP: "answer", ExecutionVersion: 2})
			})
			require.ErrorIs(t, err, failure)
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newFixture(t)
			tt.run(t, f, interviewrepository.Commands{DB: f.store.DB, Clock: f.store.Clock})
		})
	}
}
