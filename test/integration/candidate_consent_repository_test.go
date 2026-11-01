package integration_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/rfanazhari/amer-prototype/application/repository/accessport"
	"github.com/rfanazhari/amer-prototype/application/repository/candidateinterviewport"
	"github.com/rfanazhari/amer-prototype/application/service"
	"github.com/rfanazhari/amer-prototype/application/usecase/candidateinterviewusecase"
	"github.com/rfanazhari/amer-prototype/domain/aggregate/candidateinterview"
	"github.com/rfanazhari/amer-prototype/infrastructure/repository/candidateinterviewrepository"
	"github.com/rfanazhari/amer-prototype/test/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// consentFixture holds one assignment with a live candidate session on the release schema.
type consentFixture struct {
	executionFixture
	sessionID string
	store     candidateinterviewrepository.ConsentStore
}

// wallClock reports the real time for repository session checks.
type wallClock struct{}

// Now returns the current instant.
func (wallClock) Now() time.Time { return time.Now() }

// newConsentFixture adds an invitation and a two-hour candidate session to a release-schema assignment.
func newConsentFixture(t *testing.T) consentFixture {
	t.Helper()
	f := newExecutionFixture(t)
	account := insertACPAccount(t, f.db)
	invitationID, sessionID := uuid.NewString(), uuid.NewString()
	now := time.Now().UTC()
	require.NoError(t, f.db.Exec(`INSERT INTO interview_invitations (id, assignment_id, token_hash, created_by, created_at, expires_at) VALUES (?, ?, repeat('a', 64), ?, ?, ?)`,
		invitationID, f.assignmentID, account.ID, now.Add(-time.Minute), now.Add(time.Hour)).Error)
	require.NoError(t, f.db.Exec(`INSERT INTO candidate_sessions (id, assignment_id, invitation_id, token_hash, created_at, expires_at) VALUES (?, ?, ?, repeat('b', 64), ?, ?)`,
		sessionID, f.assignmentID, invitationID, now.Add(-time.Minute), now.Add(2*time.Hour)).Error)
	t.Cleanup(func() {
		require.NoError(t, f.db.Exec(`DELETE FROM interview_consents WHERE assignment_id = ?`, f.assignmentID).Error)
		require.NoError(t, f.db.Exec(`DELETE FROM candidate_sessions WHERE assignment_id = ?`, f.assignmentID).Error)
		require.NoError(t, f.db.Exec(`DELETE FROM interview_invitations WHERE assignment_id = ?`, f.assignmentID).Error)
	})
	return consentFixture{executionFixture: f, sessionID: sessionID, store: candidateinterviewrepository.ConsentStore{DB: f.db, Clock: wallClock{}}}
}

// acceptOperation wires AcceptNotice over the real store with the fixture session and notice.
func (f consentFixture) acceptOperation(t *testing.T) candidateinterviewusecase.AcceptNotice {
	t.Helper()
	current := mocks.NewCurrentCandidateSession(t)
	record := testSessionRecord(f)
	current.On("Resolve", mock.Anything).Return(record, nil).Maybe()
	catalog := mocks.NewNoticeCatalog(t)
	version, err := candidateinterview.ParseNoticeVersion("notice-261102")
	require.NoError(t, err)
	catalog.On("Current", mock.Anything).Return(service.Notice{Version: version, Statement: "statement"}, true).Maybe()
	return candidateinterviewusecase.AcceptNotice{Current: current, Catalog: catalog, Tx: f.store, Consents: f.store, Clock: wallClock{}}
}

// TestCandidateConsentRepositoryPostgres proves uniqueness, rollback, revocation and retired-version reads on PostgreSQL.
func TestCandidateConsentRepositoryPostgres(t *testing.T) {
	t.Run("concurrent accepts collapse to one row and one id", func(t *testing.T) {
		f := newConsentFixture(t)
		op := f.acceptOperation(t)
		const callers = 8
		var wg sync.WaitGroup
		results := make([]*candidateinterviewusecase.AcceptNoticeResult, callers)
		errs := make([]error, callers)
		for i := 0; i < callers; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				results[i], errs[i] = op.Execute(context.Background(), candidateinterviewusecase.AcceptNoticeInput{NoticeVersion: "notice-261102", Accepted: true})
			}()
		}
		wg.Wait()
		created := 0
		for i := range results {
			require.NoError(t, errs[i])
			assert.Equal(t, results[0].Consent.ID, results[i].Consent.ID)
			if results[i].Created {
				created++
			}
		}
		assert.Equal(t, 1, created)
		assert.Equal(t, int64(1), f.count(t, `SELECT count(*) FROM interview_consents WHERE assignment_id = ?`, f.assignmentID))
	})
	t.Run("revoked assignment writes nothing", func(t *testing.T) {
		f := newConsentFixture(t)
		require.NoError(t, f.db.Exec(`UPDATE interview_assignments SET access_revoked_at = now() WHERE id = ?`, f.assignmentID).Error)
		got, err := f.acceptOperation(t).Execute(context.Background(), candidateinterviewusecase.AcceptNoticeInput{NoticeVersion: "notice-261102", Accepted: true})
		require.ErrorIs(t, err, service.ErrCandidateSessionUnbound)
		assert.Nil(t, got)
		assert.Equal(t, int64(0), f.count(t, `SELECT count(*) FROM interview_consents WHERE assignment_id = ?`, f.assignmentID))
	})
	t.Run("callback failure rolls the insert back", func(t *testing.T) {
		f := newConsentFixture(t)
		version, err := candidateinterview.ParseNoticeVersion("notice-261102")
		require.NoError(t, err)
		snap := candidateinterview.ConsentSnapshot{
			ID: uuid.NewString(), AssignmentID: f.assignmentID, NoticeVersion: version,
			CandidateSessionID: f.sessionID, AcceptedAt: time.Now().UTC().Truncate(time.Millisecond),
		}
		probe := errors.New("probe")
		err = f.store.WithAssignment(context.Background(), f.assignmentID, f.sessionID, func(ctx context.Context) error {
			require.NoError(t, f.store.Insert(ctx, snap))
			return probe
		})
		require.ErrorIs(t, err, probe)
		assert.Equal(t, int64(0), f.count(t, `SELECT count(*) FROM interview_consents WHERE assignment_id = ?`, f.assignmentID))
	})
	t.Run("retired version stays readable and does not satisfy the current one", func(t *testing.T) {
		f := newConsentFixture(t)
		old, err := candidateinterview.ParseNoticeVersion("notice-old")
		require.NoError(t, err)
		current, err := candidateinterview.ParseNoticeVersion("notice-261102")
		require.NoError(t, err)
		snap := candidateinterview.ConsentSnapshot{
			ID: uuid.NewString(), AssignmentID: f.assignmentID, NoticeVersion: old,
			CandidateSessionID: f.sessionID, AcceptedAt: time.Now().UTC().Truncate(time.Millisecond),
		}
		require.NoError(t, f.store.WithAssignment(context.Background(), f.assignmentID, f.sessionID, func(ctx context.Context) error {
			return f.store.Insert(ctx, snap)
		}))
		got, err := f.store.Find(context.Background(), f.assignmentID, old)
		require.NoError(t, err)
		assert.Equal(t, snap.ID, got.ID)
		byID, err := f.store.FindByID(context.Background(), f.assignmentID, snap.ID)
		require.NoError(t, err)
		assert.Equal(t, snap.ID, byID.ID)
		_, err = f.store.Find(context.Background(), f.assignmentID, current)
		require.ErrorIs(t, err, candidateinterviewport.ErrNotFound)
		_, err = f.store.FindByID(context.Background(), uuid.NewString(), snap.ID)
		require.ErrorIs(t, err, candidateinterviewport.ErrNotFound)
	})
}

// testSessionRecord returns the fixture session as the cookie resolver would report it.
func testSessionRecord(f consentFixture) accessport.CandidateSessionRecord {
	now := time.Now().UTC()
	return accessport.CandidateSessionRecord{ID: f.sessionID, AssignmentID: f.assignmentID, CreatedAt: now.Add(-time.Minute), ExpiresAt: now.Add(2 * time.Hour)}
}
