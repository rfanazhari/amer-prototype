package accessusecase_test

import (
	"context"
	"testing"
	"time"

	"github.com/rfanazhari/amer-prototype/application/repository/accessport"
	"github.com/rfanazhari/amer-prototype/application/service"
	"github.com/rfanazhari/amer-prototype/application/usecase/accessusecase"
	"github.com/rfanazhari/amer-prototype/domain/enum"
	"github.com/rfanazhari/amer-prototype/test/mocks"
	"github.com/rfanazhari/amer-prototype/test/testdata"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// issueInvitationCase names one assignment-serialized invitation issuance outcome.
type issueInvitationCase struct {
	name     string
	actor    accessport.AccountRecord
	facts    accessport.AssignmentFacts
	existing []accessport.InvitationRecord
	wantErr  error
}

// TestIssueInvitationReplacesCurrentInvitationWithoutRevokingSessions encodes reissue vs assignment revocation.
func TestIssueInvitationReplacesCurrentInvitationWithoutRevokingSessions(t *testing.T) {
	hr := testdata.AdministrativeAccountRecord(t)
	hr.Role = enum.HR
	admin := testdata.AdministrativeAccountRecord(t)
	session := testdata.AdministrativeSessionRecord(t)
	live := testdata.AssignmentFacts(t)
	revoked := testdata.AssignmentFacts(t)
	revoked.AccessRevoked = true
	currentInvitation := testdata.InvitationRecord(t)
	tests := []issueInvitationCase{
		{name: "hr reissues and revokes former invitation only", actor: hr, facts: live, existing: []accessport.InvitationRecord{currentInvitation}},
		{name: "admin cannot issue invitation", actor: admin, facts: live, wantErr: service.ErrForbidden},
		{name: "revoked assignment denies invitation", actor: hr, facts: revoked, wantErr: service.ErrForbidden},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assignments := mocks.NewAssignmentAccess(t)
			current := mocks.NewCurrentAdministrativeSession(t)
			tokens := mocks.NewTokenSecrets(t)
			clock := mocks.NewClock(t)
			actorSession := session
			actorSession.AccountID = tt.actor.ID
			current.On("Resolve", mock.Anything).Return(actorSession, tt.actor, nil).Once()
			clock.On("Now").Return(testdata.Now())
			if tt.actor.Role == enum.HR {
				if tt.wantErr == nil {
					tokens.On("Issue", mock.Anything, service.CredentialInvitation).Return("invitation-secret", testdata.InvitationTokenHash, nil).Once()
				}
				assignments.On("WithAssignment", mock.Anything, testdata.AssignmentID, mock.Anything).Return(func(ctx context.Context, assignmentID string, fn func(accessport.AssignmentTx) error) error {
					tx := mocks.NewAssignmentTx(t)
					tx.On("Facts", mock.Anything).Return(tt.facts, nil).Once()
					if tt.wantErr == nil {
						tx.On("CurrentInvitations", mock.Anything).Return(tt.existing, nil).Once()
						tx.On("RevokeInvitation", mock.Anything, currentInvitation.ID, mock.MatchedBy(func(revokedAt *time.Time) bool {
							return revokedAt != nil && revokedAt.Equal(testdata.Now())
						})).Return(nil).Once()
						tx.On("InsertInvitation", mock.Anything, mock.MatchedBy(func(record accessport.InvitationRecord) bool {
							return record.AssignmentID == testdata.AssignmentID &&
								record.TokenHash == testdata.InvitationTokenHash &&
								record.CreatedBy == hr.ID
						})).Return(nil).Once()
					}
					return fn(tx)
				}).Once()
			}
			got, err := (accessusecase.IssueInvitation{
				Assignments: assignments,
				Current:     current,
				Tokens:      tokens,
				Clock:       clock,
			}).Execute(context.Background(), testdata.AssignmentID)
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				assert.Nil(t, got)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, got)
			assert.Equal(t, testdata.AssignmentID, got.AssignmentID)
			assert.Equal(t, "invitation-secret", got.Token)
			assert.Equal(t, testdata.Now().Add(24*time.Hour), got.ExpiresAt)
		})
	}
}
