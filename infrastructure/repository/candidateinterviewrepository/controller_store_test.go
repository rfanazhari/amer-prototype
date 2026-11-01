package candidateinterviewrepository_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/rfanazhari/amer-prototype/application/repository/candidateinterviewport"
	"github.com/rfanazhari/amer-prototype/domain/aggregate/candidateinterview"
	"github.com/rfanazhari/amer-prototype/infrastructure/repository/candidateinterviewrepository"
	"github.com/rfanazhari/amer-prototype/test/testdata"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// controllerExecutionID is the fixture execution controlled by the fixture tab.
const controllerExecutionID = "22222222-2222-4222-8222-222222222222"

// controllerColumns lists the interview_controllers columns scanned by reads.
var controllerColumns = []string{"assignment_id", "execution_id", "controller_id", "version", "updated_at"}

// controllerStore opens the controller adapter over sqlmock with ping disabled so tests own SQL expectations.
func controllerStore(t *testing.T) (candidateinterviewrepository.ControllerStore, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, mock.ExpectationsWereMet())
		_ = db.Close()
	})
	gdb, err := gorm.Open(postgres.New(postgres.Config{Conn: db, DriverName: "postgres"}), &gorm.Config{
		Logger:                 logger.Default.LogMode(logger.Silent),
		SkipDefaultTransaction: true,
		DisableAutomaticPing:   true,
	})
	require.NoError(t, err)
	return candidateinterviewrepository.ControllerStore{DB: gdb}, mock
}

// controllerSnapshot returns a consistent controller at the given tab and version.
func controllerSnapshot(t *testing.T, tab string, version int64) candidateinterview.ControllerSnapshot {
	t.Helper()
	id, err := candidateinterview.ParseControllerID(tab)
	require.NoError(t, err)
	return candidateinterview.ControllerSnapshot{
		AssignmentID: testdata.AssignmentID, ExecutionID: controllerExecutionID,
		ControllerID: id, Version: version, UpdatedAt: testdata.Now().Add(time.Minute),
	}
}

// expectControllerLock expects the transaction and the locking read of the controller row.
func expectControllerLock(mock sqlmock.Sqlmock, snap candidateinterview.ControllerSnapshot) {
	mock.ExpectBegin()
	mock.ExpectQuery(`(?i)SELECT .* FROM ["']?interview_controllers["']? WHERE assignment_id = .+ LIMIT .+ FOR UPDATE`).
		WithArgs(testdata.AssignmentID, 1).
		WillReturnRows(sqlmock.NewRows(controllerColumns).AddRow(snap.AssignmentID, snap.ExecutionID, snap.ControllerID.String(), snap.Version, snap.UpdatedAt))
}

// TestControllerClaimInsertsFirstController writes every column of the first controller.
func TestControllerClaimInsertsFirstController(t *testing.T) {
	store, mock := controllerStore(t)
	snap := controllerSnapshot(t, "tab-a", 1)
	mock.ExpectExec(`(?i)INSERT INTO ["']?interview_controllers["']?`).
		WithArgs(snap.AssignmentID, snap.ExecutionID, "tab-a", int64(1), snap.UpdatedAt).
		WillReturnResult(sqlmock.NewResult(1, 1))

	require.NoError(t, store.Claim(context.Background(), snap))
}

// TestControllerClaimMapsUniqueViolation reports ErrAlreadyExists and passes other failures through.
func TestControllerClaimMapsUniqueViolation(t *testing.T) {
	t.Run("unique violation", func(t *testing.T) {
		store, mock := controllerStore(t)
		mock.ExpectExec(`(?i)INSERT INTO ["']?interview_controllers["']?`).WillReturnError(&pgconn.PgError{Code: "23505"})
		err := store.Claim(context.Background(), controllerSnapshot(t, "tab-a", 1))
		require.ErrorIs(t, err, candidateinterviewport.ErrAlreadyExists)
	})
	t.Run("other failure", func(t *testing.T) {
		store, mock := controllerStore(t)
		boom := errors.New("boom")
		mock.ExpectExec(`(?i)INSERT INTO ["']?interview_controllers["']?`).WillReturnError(boom)
		err := store.Claim(context.Background(), controllerSnapshot(t, "tab-a", 1))
		require.ErrorIs(t, err, boom)
		assert.NotErrorIs(t, err, candidateinterviewport.ErrAlreadyExists)
	})
}

// TestControllerFindRestoresStoredController scopes the read to the assignment and copies the stored fields.
func TestControllerFindRestoresStoredController(t *testing.T) {
	store, mock := controllerStore(t)
	snap := controllerSnapshot(t, "tab-b", 3)
	mock.ExpectQuery(`(?i)SELECT .* FROM ["']?interview_controllers["']? WHERE assignment_id = .+ LIMIT`).
		WithArgs(testdata.AssignmentID, 1).
		WillReturnRows(sqlmock.NewRows(controllerColumns).AddRow(snap.AssignmentID, snap.ExecutionID, "tab-b", int64(3), snap.UpdatedAt))

	got, err := store.Find(context.Background(), testdata.AssignmentID)

	require.NoError(t, err)
	assert.Equal(t, snap.AssignmentID, got.AssignmentID)
	assert.Equal(t, snap.ExecutionID, got.ExecutionID)
	assert.Equal(t, "tab-b", got.ControllerID.String())
	assert.EqualValues(t, 3, got.Version)
	assert.True(t, got.UpdatedAt.Equal(snap.UpdatedAt))
}

// TestControllerFindMapsAbsenceAndCorruption reports ErrNotFound for no row and a technical error for a corrupt row.
func TestControllerFindMapsAbsenceAndCorruption(t *testing.T) {
	t.Run("absent", func(t *testing.T) {
		store, mock := controllerStore(t)
		mock.ExpectQuery(`(?i)SELECT .* FROM ["']?interview_controllers["']?`).WillReturnRows(sqlmock.NewRows(controllerColumns))
		_, err := store.Find(context.Background(), testdata.AssignmentID)
		require.ErrorIs(t, err, candidateinterviewport.ErrNotFound)
	})
	t.Run("corrupt controller id", func(t *testing.T) {
		store, mock := controllerStore(t)
		snap := controllerSnapshot(t, "tab-a", 1)
		mock.ExpectQuery(`(?i)SELECT .* FROM ["']?interview_controllers["']?`).
			WillReturnRows(sqlmock.NewRows(controllerColumns).AddRow(snap.AssignmentID, snap.ExecutionID, " bad", int64(1), snap.UpdatedAt))
		_, err := store.Find(context.Background(), testdata.AssignmentID)
		require.Error(t, err)
		assert.NotErrorIs(t, err, candidateinterviewport.ErrNotFound)
	})
}

// TestControllerUpdateLocksThenWritesExplicitFields persists id, version and updated_at after the callback succeeds.
func TestControllerUpdateLocksThenWritesExplicitFields(t *testing.T) {
	store, mock := controllerStore(t)
	current := controllerSnapshot(t, "tab-a", 1)
	next, err := candidateinterview.ParseControllerID("tab-b")
	require.NoError(t, err)
	takeoverAt := testdata.Now().Add(time.Hour)
	expectControllerLock(mock, current)
	mock.ExpectExec(`(?i)UPDATE ["']?interview_controllers["']? SET .controller_id.=.+,.version.=.+,.updated_at.=.+ WHERE assignment_id = .+`).
		WithArgs("tab-b", int64(2), takeoverAt, testdata.AssignmentID).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	err = store.Update(context.Background(), testdata.AssignmentID, func(c *candidateinterview.InterviewController) error {
		return c.Takeover(1, next, false, takeoverAt)
	})

	require.NoError(t, err)
}

// TestControllerUpdateStaleVersionLeavesRow rolls back without writing when the callback refuses the stale version.
func TestControllerUpdateStaleVersionLeavesRow(t *testing.T) {
	store, mock := controllerStore(t)
	expectControllerLock(mock, controllerSnapshot(t, "tab-a", 2))
	mock.ExpectRollback()
	next, err := candidateinterview.ParseControllerID("tab-b")
	require.NoError(t, err)

	err = store.Update(context.Background(), testdata.AssignmentID, func(c *candidateinterview.InterviewController) error {
		return c.Takeover(1, next, false, testdata.Now())
	})

	require.ErrorIs(t, err, candidateinterview.ErrStaleControllerVersion)
}

// TestControllerUpdateAbsentControllerSkipsCallback reports ErrNotFound without running the callback.
func TestControllerUpdateAbsentControllerSkipsCallback(t *testing.T) {
	store, mock := controllerStore(t)
	mock.ExpectBegin()
	mock.ExpectQuery(`(?i)SELECT .* FROM ["']?interview_controllers["']? .* FOR UPDATE`).WillReturnRows(sqlmock.NewRows(controllerColumns))
	mock.ExpectRollback()
	called := false

	err := store.Update(context.Background(), testdata.AssignmentID, func(*candidateinterview.InterviewController) error {
		called = true
		return nil
	})

	require.ErrorIs(t, err, candidateinterviewport.ErrNotFound)
	assert.False(t, called)
}

// TestControllerUpdateRejectsCorruptRow rolls back and does not run the callback for a corrupt stored id.
func TestControllerUpdateRejectsCorruptRow(t *testing.T) {
	store, mock := controllerStore(t)
	snap := controllerSnapshot(t, "tab-a", 1)
	mock.ExpectBegin()
	mock.ExpectQuery(`(?i)SELECT .* FROM ["']?interview_controllers["']? .* FOR UPDATE`).
		WillReturnRows(sqlmock.NewRows(controllerColumns).AddRow(snap.AssignmentID, snap.ExecutionID, " bad", int64(1), snap.UpdatedAt))
	mock.ExpectRollback()
	called := false

	err := store.Update(context.Background(), testdata.AssignmentID, func(*candidateinterview.InterviewController) error {
		called = true
		return nil
	})

	require.Error(t, err)
	assert.NotErrorIs(t, err, candidateinterviewport.ErrNotFound)
	assert.False(t, called)
}
