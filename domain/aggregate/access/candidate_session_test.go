package access_test

import (
	"testing"
	"time"

	"github.com/rfanazhari/amer-prototype/domain/aggregate/access"
	"github.com/rfanazhari/amer-prototype/domain/valueobject"
	"github.com/rfanazhari/amer-prototype/test/testdata"
	"github.com/stretchr/testify/require"
)

const candidateSessionID = "eeeeeeee-eeee-4eee-8eee-eeeeeeeeeee1"
const assignmentBID = "ffffffff-ffff-4fff-8fff-fffffffffff1"

// candidateSessionCreateCase defines identity inputs for a new candidate session.
type candidateSessionCreateCase struct {
	name         string
	id           string
	assignmentID string
	invitationID string
	now          time.Time
	wantErr      bool
}

// TestNewCandidateSessionSetsExactTwoHourExpiry records assignment-scoped session lifetime.
func TestNewCandidateSessionSetsExactTwoHourExpiry(t *testing.T) {
	now := testdata.Now()
	tests := []candidateSessionCreateCase{
		{"valid", candidateSessionID, assignmentAID, invitationID, now, false},
		{"empty id", "", assignmentAID, invitationID, now, true},
		{"empty assignment", candidateSessionID, "", invitationID, now, true},
		{"empty invitation", candidateSessionID, assignmentAID, "", now, true},
		{"zero time", candidateSessionID, assignmentAID, invitationID, time.Time{}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := access.NewCandidateSession(tt.id, tt.assignmentID, tt.invitationID, tt.now)
			if tt.wantErr {
				require.ErrorIs(t, err, access.ErrInvalidCandidateSession)
				require.Nil(t, got)
				return
			}
			require.NoError(t, err)
			snapshot := got.Snapshot()
			require.Equal(t, tt.assignmentID, snapshot.AssignmentID)
			require.Equal(t, tt.invitationID, snapshot.InvitationID)
			require.Equal(t, tt.now.Add(valueobject.AdministrativeDuration), snapshot.ExpiresAt)
			require.True(t, got.Current(tt.now))
			require.False(t, got.Current(tt.now.Add(valueobject.AdministrativeDuration)))
			require.True(t, got.Authorizes(tt.assignmentID, tt.now.Add(time.Minute)))
			require.False(t, got.Authorizes(assignmentBID, tt.now.Add(time.Minute)))
		})
	}
}
