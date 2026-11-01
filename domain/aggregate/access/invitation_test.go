package access_test

import (
	"testing"
	"time"

	"github.com/rfanazhari/amer-prototype/domain/aggregate/access"
	"github.com/rfanazhari/amer-prototype/domain/valueobject"
	"github.com/rfanazhari/amer-prototype/test/testdata"
	"github.com/stretchr/testify/require"
)

const invitationID = "cccccccc-cccc-4ccc-8ccc-ccccccccccc1"
const assignmentAID = "dddddddd-dddd-4ddd-8ddd-ddddddddddd1"

// invitationCreateCase defines identity inputs for a new invitation.
type invitationCreateCase struct {
	name         string
	id           string
	assignmentID string
	createdBy    string
	now          time.Time
	wantErr      bool
}

// TestNewInvitationSetsExactTwentyFourHourExpiry records assignment-scoped invitation lifetime.
func TestNewInvitationSetsExactTwentyFourHourExpiry(t *testing.T) {
	now := testdata.Now()
	tests := []invitationCreateCase{
		{"valid", invitationID, assignmentAID, creatorID, now, false},
		{"empty id", "", assignmentAID, creatorID, now, true},
		{"empty assignment", invitationID, "", creatorID, now, true},
		{"empty creator", invitationID, assignmentAID, "", now, true},
		{"zero time", invitationID, assignmentAID, creatorID, time.Time{}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := access.NewInvitation(tt.id, tt.assignmentID, tt.createdBy, tt.now)
			if tt.wantErr {
				require.ErrorIs(t, err, access.ErrInvalidInvitation)
				require.Nil(t, got)
				return
			}
			require.NoError(t, err)
			snapshot := got.Snapshot()
			require.Equal(t, tt.id, snapshot.ID)
			require.Equal(t, tt.assignmentID, snapshot.AssignmentID)
			require.Equal(t, tt.now.Add(valueobject.InvitationDuration), snapshot.ExpiresAt)
			require.Nil(t, snapshot.RevokedAt)
			require.True(t, got.Current(tt.now))
			require.False(t, got.Current(tt.now.Add(valueobject.InvitationDuration)))
			require.Equal(t, tt.assignmentID, got.AssignmentID())
		})
	}
}

// TestInvitationRevocationCopiesPointersAndDeniesExchange proves reissue revocation without mutating aliases.
func TestInvitationRevocationCopiesPointersAndDeniesExchange(t *testing.T) {
	now := testdata.Now()
	invitation, err := access.NewInvitation(invitationID, assignmentAID, creatorID, now)
	require.NoError(t, err)
	revokedAt := now.Add(time.Minute)
	require.NoError(t, invitation.Revoke(revokedAt))
	require.False(t, invitation.Current(revokedAt))
	snapshot := invitation.Snapshot()
	require.Equal(t, revokedAt, *snapshot.RevokedAt)
	*snapshot.RevokedAt = now.Add(time.Hour)
	require.Equal(t, revokedAt, *invitation.Snapshot().RevokedAt)
	require.ErrorIs(t, invitation.Revoke(revokedAt.Add(time.Minute)), access.ErrInvalidInvitation)
}
