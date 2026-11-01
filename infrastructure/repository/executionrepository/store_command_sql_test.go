package executionrepository_test

import (
	"context"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/rfanazhari/amer-prototype/application/repository/executionport"
	"github.com/rfanazhari/amer-prototype/infrastructure/repository/executionrepository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	commandScope       = "dddddddd-dddd-4ddd-8ddd-ddddddddddd1"
	commandKind        = "start"
	commandKey         = "start-001"
	commandFingerprint = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	otherFingerprint   = "fedcba9876543210fedcba9876543210fedcba9876543210fedcba9876543210"
	storedExecutionID  = "ffffffff-ffff-4fff-8fff-fffffffffff2"
)

// commandColumns lists the agent_execution_commands columns selected on a replay read.
var commandColumns = []string{"id", "scope", "command_kind", "request_key", "intent_fingerprint", "execution_id", "created_at"}

// expectCommandInsert expects the idempotent command insert and reports how many rows it wrote.
func expectCommandInsert(mock sqlmock.Sqlmock, affected int64) {
	mock.ExpectExec(`INSERT INTO "agent_execution_commands" .* ON CONFLICT DO NOTHING`).WithArgs(
		sqlmock.AnyArg(), commandScope, commandKind, commandKey, commandFingerprint, executionID, sqlmock.AnyArg(),
	).WillReturnResult(sqlmock.NewResult(0, affected))
}

// expectCommandRead expects the replay read of the stored command identity.
func expectCommandRead(mock sqlmock.Sqlmock, fingerprint string) {
	mock.ExpectQuery(`SELECT .* FROM "agent_execution_commands" WHERE scope = \$1 AND command_kind = \$2 AND request_key = \$3`).
		WithArgs(commandScope, commandKind, commandKey, 1).
		WillReturnRows(sqlmock.NewRows(commandColumns).AddRow("c1", commandScope, commandKind, commandKey, fingerprint, storedExecutionID, createdAt))
}

// TestExecutionStoreFindOrInsert covers the idempotent command identity over sqlmock.
func TestExecutionStoreFindOrInsert(t *testing.T) {
	tests := []storeCase{
		{
			name: "a new identity is inserted once and points at the supplied execution",
			run: func(t *testing.T, store executionrepository.Store, mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				expectCommandInsert(mock, 1)
				mock.ExpectCommit()
				id, err := store.FindOrInsert(context.Background(), commandScope, commandKind, commandKey, commandFingerprint, executionID)
				require.NoError(t, err)
				assert.Equal(t, executionID, id)
			},
		},
		{
			name: "a replay with the same fingerprint returns the stored execution without a second row",
			run: func(t *testing.T, store executionrepository.Store, mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				expectCommandInsert(mock, 0)
				expectCommandRead(mock, commandFingerprint)
				mock.ExpectCommit()
				id, err := store.FindOrInsert(context.Background(), commandScope, commandKind, commandKey, commandFingerprint, executionID)
				require.NoError(t, err)
				assert.Equal(t, storedExecutionID, id)
			},
		},
		{
			name: "a replay with another fingerprint is an idempotency conflict and rolls back",
			run: func(t *testing.T, store executionrepository.Store, mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				expectCommandInsert(mock, 0)
				expectCommandRead(mock, otherFingerprint)
				mock.ExpectRollback()
				_, err := store.FindOrInsert(context.Background(), commandScope, commandKind, commandKey, commandFingerprint, executionID)
				assert.ErrorIs(t, err, executionport.ErrIdempotencyConflict)
			},
		},
		{
			name: "a missing execution is a dependency failure and rolls back",
			run: func(t *testing.T, store executionrepository.Store, mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectExec(`INSERT INTO "agent_execution_commands"`).WillReturnError(&pgconn.PgError{Code: "23503"})
				mock.ExpectRollback()
				_, err := store.FindOrInsert(context.Background(), commandScope, commandKind, commandKey, commandFingerprint, executionID)
				assert.ErrorIs(t, err, executionport.ErrDependencyUnavailable)
			},
		},
		{
			name: "the command write joins the transaction handed by the caller",
			run: func(t *testing.T, store executionrepository.Store, mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				expectCommandInsert(mock, 1)
				mock.ExpectCommit()
				err := store.WithTransaction(context.Background(), func(ctx context.Context) error {
					_, err := store.FindOrInsert(ctx, commandScope, commandKind, commandKey, commandFingerprint, executionID)
					return err
				})
				require.NoError(t, err)
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
