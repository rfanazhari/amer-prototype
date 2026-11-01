package accessusecase_test

import (
	"context"
	"testing"
	"time"

	"github.com/rfanazhari/amer-prototype/application/repository/accessport"
	"github.com/rfanazhari/amer-prototype/application/service"
	"github.com/rfanazhari/amer-prototype/application/usecase/accessusecase"
	"github.com/rfanazhari/amer-prototype/test/mocks"
	"github.com/rfanazhari/amer-prototype/test/testdata"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// exchangeCase names one invitation-exchange outcome including retry/device policy.
type exchangeCase struct {
	name      string
	lookup    accessport.InvitationRecord
	lookupErr error
	facts     accessport.AssignmentFacts
	now       time.Time
	wantErr   error
}

// TestExchangeInvitationCreatesNewSessionWithoutExecutionSideEffects covers reuse, expiry and assignment revocation.
func TestExchangeInvitationCreatesNewSessionWithoutExecutionSideEffects(t *testing.T) {
	invitation := testdata.InvitationRecord(t)
	revokedInvitation := testdata.InvitationRecord(t)
	revokedAt := testdata.Now().Add(time.Minute)
	revokedInvitation.RevokedAt = &revokedAt
	live := testdata.AssignmentFacts(t)
	assignmentRevoked := testdata.AssignmentFacts(t)
	assignmentRevoked.AccessRevoked = true
	completed := testdata.AssignmentFacts(t)
	completed.TerminalCompleted = true
	tests := []exchangeCase{
		{name: "valid invitation creates a new session", lookup: invitation, facts: live, now: testdata.Now().Add(time.Minute)},
		{name: "repeat exchange creates another session", lookup: invitation, facts: live, now: testdata.Now().Add(2 * time.Minute)},
		{name: "completed assignment still exchanges without starting execution", lookup: invitation, facts: completed, now: testdata.Now().Add(time.Minute)},
		{name: "unknown invitation is unavailable", lookupErr: accessport.ErrNotFound, now: testdata.Now(), wantErr: accessport.ErrInvitationUnavailable},
		{name: "revoked invitation is unavailable", lookup: revokedInvitation, now: testdata.Now().Add(time.Minute), wantErr: accessport.ErrInvitationUnavailable},
		{name: "expired invitation is unavailable", lookup: invitation, now: testdata.Now().Add(24 * time.Hour), wantErr: accessport.ErrInvitationUnavailable},
		{name: "revoked assignment is unavailable", lookup: invitation, facts: assignmentRevoked, now: testdata.Now().Add(time.Minute), wantErr: accessport.ErrInvitationUnavailable},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			invitations := mocks.NewInvitations(t)
			candidates := mocks.NewCandidates(t)
			assignments := mocks.NewAssignmentAccess(t)
			tokens := mocks.NewTokenSecrets(t)
			clock := mocks.NewClock(t)
			tokens.On("Digest", service.CredentialInvitation, "invitation-secret").Return(testdata.InvitationTokenHash, nil).Once()
			clock.On("Now").Return(tt.now)
			if tt.lookupErr != nil {
				invitations.On("FindByTokenHash", mock.Anything, testdata.InvitationTokenHash).Return(accessport.InvitationRecord{}, tt.lookupErr).Once()
			} else {
				invitations.On("FindByTokenHash", mock.Anything, testdata.InvitationTokenHash).Return(tt.lookup, nil).Once()
			}
			lockAssignment := tt.lookupErr == nil && tt.lookup.RevokedAt == nil && !tt.now.Before(testdata.Now()) && (tt.wantErr == nil || tt.facts.AccessRevoked)
			if tt.now.Equal(testdata.Now().Add(24 * time.Hour)) {
				lockAssignment = false
			}
			if tt.lookup.RevokedAt != nil {
				lockAssignment = false
			}
			if lockAssignment {
				if tt.wantErr == nil {
					tokens.On("Issue", mock.Anything, service.CredentialCandidate).Return("candidate-session-secret", testdata.TokenHash, nil).Once()
					candidates.On("FindByID", mock.Anything, testdata.CandidateID).Return(testdata.CandidateRecord(t), nil).Once()
				}
				assignments.On("WithAssignment", mock.Anything, testdata.AssignmentID, mock.Anything).Return(func(ctx context.Context, assignmentID string, fn func(accessport.AssignmentTx) error) error {
					tx := mocks.NewAssignmentTx(t)
					tx.On("Facts", mock.Anything).Return(tt.facts, nil).Once()
					if tt.wantErr == nil {
						tx.On("InsertCandidateSession", mock.Anything, mock.MatchedBy(func(record accessport.CandidateSessionRecord) bool {
							return record.AssignmentID == testdata.AssignmentID &&
								record.InvitationID == testdata.InvitationID &&
								record.TokenHash == testdata.TokenHash
						})).Return(nil).Once()
					}
					return fn(tx)
				}).Once()
			}
			got, err := (accessusecase.ExchangeInvitation{
				Invitations: invitations,
				Candidates:  candidates,
				Assignments: assignments,
				Tokens:      tokens,
				Clock:       clock,
			}).Execute(context.Background(), "invitation-secret")
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				assert.Nil(t, got)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, got)
			assert.Equal(t, testdata.CandidateID, got.Candidate.ID)
			assert.Equal(t, testdata.AssignmentID, got.AssignmentID)
			assert.Equal(t, "candidate-session-secret", got.SessionToken)
			assert.Empty(t, got.CSRFToken)
			assert.Equal(t, tt.now.Add(2*time.Hour), got.Session.ExpiresAt)
		})
	}
}
