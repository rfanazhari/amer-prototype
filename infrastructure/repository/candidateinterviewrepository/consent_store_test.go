package candidateinterviewrepository_test

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/rfanazhari/amer-prototype/application/repository/candidateinterviewport"
	"github.com/rfanazhari/amer-prototype/domain/aggregate/candidateinterview"
	"github.com/rfanazhari/amer-prototype/infrastructure/repository/candidateinterviewrepository"
	"github.com/rfanazhari/amer-prototype/test/mocks"
	"github.com/rfanazhari/amer-prototype/test/testdata"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// consentStore opens the consent adapter over sqlmock with ping disabled so tests own SQL expectations.
func consentStore(t *testing.T) (candidateinterviewrepository.ConsentStore, sqlmock.Sqlmock) {
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
	clock := mocks.NewClock(t)
	clock.On("Now").Return(testdata.Now()).Maybe()
	return candidateinterviewrepository.ConsentStore{DB: gdb, Clock: clock}, mock
}

// snapshot returns a consistent acceptance of the fixture version.
func snapshot(t *testing.T) candidateinterview.ConsentSnapshot {
	t.Helper()
	v, err := candidateinterview.ParseNoticeVersion("notice-261102")
	require.NoError(t, err)
	return candidateinterview.ConsentSnapshot{
		ID: "11111111-1111-4111-8111-111111111111", AssignmentID: testdata.AssignmentID, NoticeVersion: v,
		CandidateSessionID: testdata.CandidateSessionID, AcceptedAt: testdata.Now().Add(time.Minute),
	}
}

// consentColumns lists the interview_consents columns scanned by reads.
var consentColumns = []string{"id", "assignment_id", "notice_version", "accepted_at", "candidate_session_id"}

// expectLockAndSession expects the assignment lock followed by the candidate session re-read.
func expectLockAndSession(mock sqlmock.Sqlmock, revokedAssignment *time.Time, sessionExpires time.Time, sessionRevoked *time.Time) {
	mock.ExpectBegin()
	mock.ExpectQuery(`(?i)SELECT access_revoked_at FROM interview_assignments WHERE id = .+ FOR UPDATE`).
		WithArgs(testdata.AssignmentID).
		WillReturnRows(sqlmock.NewRows([]string{"access_revoked_at"}).AddRow(revokedAssignment))
	mock.ExpectQuery(`(?i)SELECT assignment_id, expires_at, revoked_at FROM candidate_sessions WHERE id = .+`).
		WithArgs(testdata.CandidateSessionID).
		WillReturnRows(sqlmock.NewRows([]string{"assignment_id", "expires_at", "revoked_at"}).AddRow(testdata.AssignmentID, sessionExpires, sessionRevoked))
}

// TestWithAssignmentLocksThenInsertsAndCommits covers the FOR UPDATE lock first, the joined insert and the commit.
func TestWithAssignmentLocksThenInsertsAndCommits(t *testing.T) {
	store, mock := consentStore(t)
	snap := snapshot(t)
	expectLockAndSession(mock, nil, testdata.Now().Add(time.Hour), nil)
	mock.ExpectExec(`(?i)INSERT INTO ["']?interview_consents["']?`).
		WithArgs(snap.ID, snap.AssignmentID, "notice-261102", snap.AcceptedAt, snap.CandidateSessionID).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	err := store.WithAssignment(context.Background(), testdata.AssignmentID, testdata.CandidateSessionID, func(ctx context.Context) error {
		return store.Insert(ctx, snap)
	})

	require.NoError(t, err)
}

// TestWithAssignmentRefusesUnavailableAssignment skips the callback and rolls back for revocation and ended sessions.
func TestWithAssignmentRefusesUnavailableAssignment(t *testing.T) {
	revoked := testdata.Now().Add(-time.Minute)
	// unavailableCase is one reason the assignment boundary refuses the callback.
	type unavailableCase struct {
		name       string
		assignment *time.Time
		expires    time.Time
		revoked    *time.Time
	}
	tests := []unavailableCase{
		{name: "assignment access revoked", assignment: &revoked, expires: testdata.Now().Add(time.Hour)},
		{name: "session revoked", expires: testdata.Now().Add(time.Hour), revoked: &revoked},
		{name: "session expiry equals now", expires: testdata.Now()},
		{name: "session expired", expires: testdata.Now().Add(-time.Second)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store, mock := consentStore(t)
			expectLockAndSession(mock, tt.assignment, tt.expires, tt.revoked)
			mock.ExpectRollback()
			called := false
			err := store.WithAssignment(context.Background(), testdata.AssignmentID, testdata.CandidateSessionID, func(context.Context) error {
				called = true
				return nil
			})
			require.ErrorIs(t, err, candidateinterviewport.ErrAssignmentUnavailable)
			assert.False(t, called)
		})
	}
}

// TestWithAssignmentMissingRowsAndForeignSession refuses an absent assignment, an absent session and another assignment's session.
func TestWithAssignmentMissingRowsAndForeignSession(t *testing.T) {
	t.Run("absent assignment", func(t *testing.T) {
		store, mock := consentStore(t)
		mock.ExpectBegin()
		mock.ExpectQuery(`(?i)SELECT access_revoked_at FROM interview_assignments`).WithArgs(testdata.AssignmentID).WillReturnError(sql.ErrNoRows)
		mock.ExpectRollback()
		err := store.WithAssignment(context.Background(), testdata.AssignmentID, testdata.CandidateSessionID, func(context.Context) error { return nil })
		require.ErrorIs(t, err, candidateinterviewport.ErrAssignmentUnavailable)
	})
	t.Run("absent session", func(t *testing.T) {
		store, mock := consentStore(t)
		mock.ExpectBegin()
		mock.ExpectQuery(`(?i)SELECT access_revoked_at FROM interview_assignments`).WithArgs(testdata.AssignmentID).
			WillReturnRows(sqlmock.NewRows([]string{"access_revoked_at"}).AddRow(nil))
		mock.ExpectQuery(`(?i)SELECT assignment_id, expires_at, revoked_at FROM candidate_sessions`).WithArgs(testdata.CandidateSessionID).WillReturnError(sql.ErrNoRows)
		mock.ExpectRollback()
		err := store.WithAssignment(context.Background(), testdata.AssignmentID, testdata.CandidateSessionID, func(context.Context) error { return nil })
		require.ErrorIs(t, err, candidateinterviewport.ErrAssignmentUnavailable)
	})
	t.Run("session of another assignment", func(t *testing.T) {
		store, mock := consentStore(t)
		mock.ExpectBegin()
		mock.ExpectQuery(`(?i)SELECT access_revoked_at FROM interview_assignments`).WithArgs(testdata.AssignmentID).
			WillReturnRows(sqlmock.NewRows([]string{"access_revoked_at"}).AddRow(nil))
		mock.ExpectQuery(`(?i)SELECT assignment_id, expires_at, revoked_at FROM candidate_sessions`).WithArgs(testdata.CandidateSessionID).
			WillReturnRows(sqlmock.NewRows([]string{"assignment_id", "expires_at", "revoked_at"}).AddRow("ffffffff-ffff-4fff-8fff-ffffffffffff", testdata.Now().Add(time.Hour), nil))
		mock.ExpectRollback()
		err := store.WithAssignment(context.Background(), testdata.AssignmentID, testdata.CandidateSessionID, func(context.Context) error { return nil })
		require.ErrorIs(t, err, candidateinterviewport.ErrAssignmentUnavailable)
	})
}

// TestWithAssignmentRollsBackWhenCallbackFails returns the callback error after rollback.
func TestWithAssignmentRollsBackWhenCallbackFails(t *testing.T) {
	store, mock := consentStore(t)
	boom := errors.New("boom")
	expectLockAndSession(mock, nil, testdata.Now().Add(time.Hour), nil)
	mock.ExpectRollback()
	err := store.WithAssignment(context.Background(), testdata.AssignmentID, testdata.CandidateSessionID, func(context.Context) error { return boom })
	require.ErrorIs(t, err, boom)
}

// TestInsertMapsUniqueViolation reports ErrAlreadyExists and rolls the transaction back.
func TestInsertMapsUniqueViolation(t *testing.T) {
	store, mock := consentStore(t)
	snap := snapshot(t)
	expectLockAndSession(mock, nil, testdata.Now().Add(time.Hour), nil)
	mock.ExpectExec(`(?i)INSERT INTO ["']?interview_consents["']?`).WillReturnError(&pgconn.PgError{Code: "23505"})
	mock.ExpectRollback()

	err := store.WithAssignment(context.Background(), testdata.AssignmentID, testdata.CandidateSessionID, func(ctx context.Context) error {
		return store.Insert(ctx, snap)
	})

	require.ErrorIs(t, err, candidateinterviewport.ErrAlreadyExists)
}

// TestFindReadsByAssignmentAndVersion restores the stored acceptance, including a retired version.
func TestFindReadsByAssignmentAndVersion(t *testing.T) {
	store, mock := consentStore(t)
	snap := snapshot(t)
	mock.ExpectQuery(`(?i)SELECT .* FROM ["']?interview_consents["']? WHERE assignment_id = .+ AND notice_version = .+ LIMIT`).
		WithArgs(testdata.AssignmentID, "notice-261102", 1).
		WillReturnRows(sqlmock.NewRows(consentColumns).AddRow(snap.ID, snap.AssignmentID, "notice-261102", snap.AcceptedAt, snap.CandidateSessionID))

	got, err := store.Find(context.Background(), testdata.AssignmentID, snap.NoticeVersion)

	require.NoError(t, err)
	assert.Equal(t, snap.ID, got.ID)
	assert.Equal(t, "notice-261102", got.NoticeVersion.String())
	assert.True(t, got.AcceptedAt.Equal(snap.AcceptedAt))
	assert.Equal(t, snap.CandidateSessionID, got.CandidateSessionID)
}

// TestFindMapsAbsenceAndCorruption reports ErrNotFound for no row and a technical error for a corrupt row.
func TestFindMapsAbsenceAndCorruption(t *testing.T) {
	t.Run("absent", func(t *testing.T) {
		store, mock := consentStore(t)
		mock.ExpectQuery(`(?i)SELECT .* FROM ["']?interview_consents["']?`).WillReturnRows(sqlmock.NewRows(consentColumns))
		_, err := store.Find(context.Background(), testdata.AssignmentID, snapshot(t).NoticeVersion)
		require.ErrorIs(t, err, candidateinterviewport.ErrNotFound)
	})
	t.Run("corrupt version", func(t *testing.T) {
		store, mock := consentStore(t)
		snap := snapshot(t)
		mock.ExpectQuery(`(?i)SELECT .* FROM ["']?interview_consents["']?`).
			WillReturnRows(sqlmock.NewRows(consentColumns).AddRow(snap.ID, snap.AssignmentID, " bad", snap.AcceptedAt, snap.CandidateSessionID))
		_, err := store.Find(context.Background(), testdata.AssignmentID, snap.NoticeVersion)
		require.Error(t, err)
		assert.NotErrorIs(t, err, candidateinterviewport.ErrNotFound)
	})
}

// TestFindByIDScopesToAssignment looks the acceptance up by id inside the assignment only.
func TestFindByIDScopesToAssignment(t *testing.T) {
	store, mock := consentStore(t)
	snap := snapshot(t)
	mock.ExpectQuery(`(?i)SELECT .* FROM ["']?interview_consents["']? WHERE id = .+ AND assignment_id = .+ LIMIT`).
		WithArgs(snap.ID, testdata.AssignmentID, 1).
		WillReturnRows(sqlmock.NewRows(consentColumns).AddRow(snap.ID, snap.AssignmentID, "notice-261102", snap.AcceptedAt, snap.CandidateSessionID))

	got, err := store.FindByID(context.Background(), testdata.AssignmentID, snap.ID)

	require.NoError(t, err)
	assert.Equal(t, snap.ID, got.ID)
}
