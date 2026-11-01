package candidateinterviewusecase_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/rfanazhari/amer-prototype/application/repository/candidateinterviewport"
	"github.com/rfanazhari/amer-prototype/application/service"
	"github.com/rfanazhari/amer-prototype/application/usecase/candidateinterviewusecase"
	"github.com/rfanazhari/amer-prototype/domain/aggregate/candidateinterview"
	"github.com/rfanazhari/amer-prototype/test/mocks"
	"github.com/rfanazhari/amer-prototype/test/testdata"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// currentVersion is the fixture catalog version.
const currentVersion = "notice-261102"

// acceptGraph holds the mocked ports of one AcceptNotice scenario.
type acceptGraph struct {
	current  *mocks.CurrentCandidateSession
	catalog  *mocks.NoticeCatalog
	tx       *mocks.ConsentTransaction
	consents *mocks.Consents
	clock    *mocks.Clock
}

// newAcceptGraph builds strict mocks for AcceptNotice.
func newAcceptGraph(t *testing.T) acceptGraph {
	t.Helper()
	return acceptGraph{
		current:  mocks.NewCurrentCandidateSession(t),
		catalog:  mocks.NewNoticeCatalog(t),
		tx:       mocks.NewConsentTransaction(t),
		consents: mocks.NewConsents(t),
		clock:    mocks.NewClock(t),
	}
}

// operation returns AcceptNotice wired to the mocks.
func (g acceptGraph) operation() candidateinterviewusecase.AcceptNotice {
	return candidateinterviewusecase.AcceptNotice{Current: g.current, Catalog: g.catalog, Tx: g.tx, Consents: g.consents, Clock: g.clock}
}

// runInTx makes the transaction mock run the callback with the supplied context.
func runInTx(_ context.Context, _ string, _ string, fn func(context.Context) error) error {
	return fn(context.Background())
}

// catalogWith returns the fixture notice catalog entry.
func catalogWith(t *testing.T) service.Notice {
	t.Helper()
	v, err := candidateinterview.ParseNoticeVersion(currentVersion)
	require.NoError(t, err)
	return service.Notice{Version: v, Statement: "statement"}
}

// storedConsent returns the winner's stored acceptance.
func storedConsent(t *testing.T, version string) candidateinterview.ConsentSnapshot {
	t.Helper()
	v, err := candidateinterview.ParseNoticeVersion(version)
	require.NoError(t, err)
	return candidateinterview.ConsentSnapshot{
		ID: "11111111-1111-4111-8111-111111111111", AssignmentID: testdata.AssignmentID, NoticeVersion: v,
		CandidateSessionID: testdata.CandidateSessionID, AcceptedAt: testdata.Now().Add(time.Minute),
	}
}

// TestAcceptNoticeInsertsFirstAcceptanceWithMillisecondTime stores one row for the current version.
func TestAcceptNoticeInsertsFirstAcceptanceWithMillisecondTime(t *testing.T) {
	g := newAcceptGraph(t)
	g.current.On("Resolve", mock.Anything).Return(testdata.CandidateSessionRecord(t), nil).Once()
	g.catalog.On("Current", mock.Anything).Return(catalogWith(t), true).Once()
	g.clock.On("Now").Return(testdata.Now().Add(123456789 * time.Nanosecond)).Once()
	g.tx.On("WithAssignment", mock.Anything, testdata.AssignmentID, testdata.CandidateSessionID, mock.Anything).Return(runInTx).Once()
	g.consents.On("Find", mock.Anything, testdata.AssignmentID, mock.Anything).Return(candidateinterview.ConsentSnapshot{}, candidateinterviewport.ErrNotFound).Once()
	g.consents.On("Insert", mock.Anything, mock.MatchedBy(func(s candidateinterview.ConsentSnapshot) bool {
		_, idErr := uuid.Parse(s.ID)
		return idErr == nil && s.AssignmentID == testdata.AssignmentID && s.NoticeVersion.String() == currentVersion &&
			s.CandidateSessionID == testdata.CandidateSessionID &&
			s.AcceptedAt.Equal(testdata.Now().Add(123*time.Millisecond)) && s.AcceptedAt.Location() == time.UTC
	})).Return(nil).Once()

	got, err := g.operation().Execute(context.Background(), candidateinterviewusecase.AcceptNoticeInput{NoticeVersion: currentVersion, Accepted: true})

	require.NoError(t, err)
	assert.True(t, got.Created)
	assert.Equal(t, currentVersion, got.Consent.NoticeVersion)
	assert.True(t, got.Consent.AcceptedAt.Equal(testdata.Now().Add(123*time.Millisecond)))
	assert.NotEmpty(t, got.Consent.ID)
}

// TestAcceptNoticeRetryReturnsStoredAcceptance returns the original row without a second insert.
func TestAcceptNoticeRetryReturnsStoredAcceptance(t *testing.T) {
	g := newAcceptGraph(t)
	stored := storedConsent(t, currentVersion)
	g.current.On("Resolve", mock.Anything).Return(testdata.CandidateSessionRecord(t), nil).Once()
	g.catalog.On("Current", mock.Anything).Return(catalogWith(t), true).Once()
	g.clock.On("Now").Return(testdata.Now()).Once()
	g.tx.On("WithAssignment", mock.Anything, testdata.AssignmentID, testdata.CandidateSessionID, mock.Anything).Return(runInTx).Once()
	g.consents.On("Find", mock.Anything, testdata.AssignmentID, mock.Anything).Return(stored, nil).Once()

	got, err := g.operation().Execute(context.Background(), candidateinterviewusecase.AcceptNoticeInput{NoticeVersion: currentVersion, Accepted: true})

	require.NoError(t, err)
	assert.False(t, got.Created)
	assert.Equal(t, stored.ID, got.Consent.ID)
	assert.True(t, got.Consent.AcceptedAt.Equal(stored.AcceptedAt))
}

// TestAcceptNoticeUniqueRaceReadsBackInNewTransaction returns the winner after the loser's transaction aborts.
func TestAcceptNoticeUniqueRaceReadsBackInNewTransaction(t *testing.T) {
	g := newAcceptGraph(t)
	winner := storedConsent(t, currentVersion)
	g.current.On("Resolve", mock.Anything).Return(testdata.CandidateSessionRecord(t), nil).Once()
	g.catalog.On("Current", mock.Anything).Return(catalogWith(t), true).Once()
	g.clock.On("Now").Return(testdata.Now()).Once()
	g.tx.On("WithAssignment", mock.Anything, testdata.AssignmentID, testdata.CandidateSessionID, mock.Anything).Return(runInTx).Twice()
	g.consents.On("Find", mock.Anything, testdata.AssignmentID, mock.Anything).Return(candidateinterview.ConsentSnapshot{}, candidateinterviewport.ErrNotFound).Once()
	g.consents.On("Insert", mock.Anything, mock.Anything).Return(candidateinterviewport.ErrAlreadyExists).Once()
	g.consents.On("Find", mock.Anything, testdata.AssignmentID, mock.Anything).Return(winner, nil).Once()

	got, err := g.operation().Execute(context.Background(), candidateinterviewusecase.AcceptNoticeInput{NoticeVersion: currentVersion, Accepted: true})

	require.NoError(t, err)
	assert.False(t, got.Created)
	assert.Equal(t, winner.ID, got.Consent.ID)
}

// TestAcceptNoticeEmptyCatalogWritesNothing reports the dependency failure before any transaction.
func TestAcceptNoticeEmptyCatalogWritesNothing(t *testing.T) {
	g := newAcceptGraph(t)
	g.current.On("Resolve", mock.Anything).Return(testdata.CandidateSessionRecord(t), nil).Once()
	g.catalog.On("Current", mock.Anything).Return(service.Notice{}, false).Once()

	got, err := g.operation().Execute(context.Background(), candidateinterviewusecase.AcceptNoticeInput{NoticeVersion: currentVersion, Accepted: true})

	require.ErrorIs(t, err, service.ErrInterviewDependencyUnavailable)
	assert.Nil(t, got)
}

// TestAcceptNoticeRefusals covers rejected input and policy before any write.
func TestAcceptNoticeRefusals(t *testing.T) {
	// refusalCase is one rejected acceptance.
	type refusalCase struct {
		name    string
		in      candidateinterviewusecase.AcceptNoticeInput
		session bool
		catalog bool
		wantErr error
	}
	tests := []refusalCase{
		{name: "accepted false is malformed input", in: candidateinterviewusecase.AcceptNoticeInput{NoticeVersion: currentVersion}, wantErr: candidateinterview.ErrInvalidConsent},
		{name: "version breaking the character rule", in: candidateinterviewusecase.AcceptNoticeInput{NoticeVersion: " notice", Accepted: true}, wantErr: candidateinterview.ErrInvalidNoticeVersion},
		{name: "version other than current", in: candidateinterviewusecase.AcceptNoticeInput{NoticeVersion: "notice-old", Accepted: true}, session: true, catalog: true, wantErr: candidateinterview.ErrNoticeNotCurrent},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := newAcceptGraph(t)
			if tt.session {
				g.current.On("Resolve", mock.Anything).Return(testdata.CandidateSessionRecord(t), nil).Once()
			}
			if tt.catalog {
				g.catalog.On("Current", mock.Anything).Return(catalogWith(t), true).Once()
				g.clock.On("Now").Return(testdata.Now()).Maybe()
			}
			got, err := g.operation().Execute(context.Background(), tt.in)
			require.ErrorIs(t, err, tt.wantErr)
			assert.Nil(t, got)
		})
	}
}

// TestAcceptNoticeSessionFailures maps an unbound session and an unavailable assignment to the session failure.
func TestAcceptNoticeSessionFailures(t *testing.T) {
	t.Run("unbound session stops before the catalog", func(t *testing.T) {
		g := newAcceptGraph(t)
		g.current.On("Resolve", mock.Anything).Return(testdata.CandidateSessionRecord(t), service.ErrCandidateSessionUnbound).Once()
		got, err := g.operation().Execute(context.Background(), candidateinterviewusecase.AcceptNoticeInput{NoticeVersion: currentVersion, Accepted: true})
		require.ErrorIs(t, err, service.ErrCandidateSessionUnbound)
		assert.Nil(t, got)
	})
	t.Run("revoked assignment or ended session", func(t *testing.T) {
		g := newAcceptGraph(t)
		g.current.On("Resolve", mock.Anything).Return(testdata.CandidateSessionRecord(t), nil).Once()
		g.catalog.On("Current", mock.Anything).Return(catalogWith(t), true).Once()
		g.clock.On("Now").Return(testdata.Now()).Once()
		g.tx.On("WithAssignment", mock.Anything, testdata.AssignmentID, testdata.CandidateSessionID, mock.Anything).Return(candidateinterviewport.ErrAssignmentUnavailable).Once()
		got, err := g.operation().Execute(context.Background(), candidateinterviewusecase.AcceptNoticeInput{NoticeVersion: currentVersion, Accepted: true})
		require.ErrorIs(t, err, service.ErrCandidateSessionUnbound)
		assert.Nil(t, got)
	})
	t.Run("storage failure passes through unclassified", func(t *testing.T) {
		g := newAcceptGraph(t)
		boom := errors.New("boom")
		g.current.On("Resolve", mock.Anything).Return(testdata.CandidateSessionRecord(t), nil).Once()
		g.catalog.On("Current", mock.Anything).Return(catalogWith(t), true).Once()
		g.clock.On("Now").Return(testdata.Now()).Once()
		g.tx.On("WithAssignment", mock.Anything, testdata.AssignmentID, testdata.CandidateSessionID, mock.Anything).Return(boom).Once()
		got, err := g.operation().Execute(context.Background(), candidateinterviewusecase.AcceptNoticeInput{NoticeVersion: currentVersion, Accepted: true})
		require.ErrorIs(t, err, boom)
		assert.Nil(t, got)
	})
}
