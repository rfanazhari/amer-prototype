package access_test

import (
	"testing"
	"time"

	"github.com/rfanazhari/amer-prototype/domain/aggregate/access"
	"github.com/rfanazhari/amer-prototype/test/testdata"
	"github.com/stretchr/testify/require"
)

const candidateID = "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaa1"
const creatorID = "bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbb1"

// candidateCreateCase defines identity inputs for a new candidate.
type candidateCreateCase struct {
	name      string
	id        string
	person    string
	email     string
	createdBy string
	now       time.Time
	wantErr   bool
}

// TestNewCandidateCreatesReusableIdentityWithoutCredentials records stable person identity without a login account.
func TestNewCandidateCreatesReusableIdentityWithoutCredentials(t *testing.T) {
	now := testdata.Now()
	tests := []candidateCreateCase{
		{"valid", candidateID, "Andi Pratama", "andi@example.com", creatorID, now, false},
		{"empty id", "", "Andi Pratama", "andi@example.com", creatorID, now, true},
		{"empty name", candidateID, "", "andi@example.com", creatorID, now, true},
		{"empty email", candidateID, "Andi Pratama", "", creatorID, now, true},
		{"empty creator", candidateID, "Andi Pratama", "andi@example.com", "", now, true},
		{"zero time", candidateID, "Andi Pratama", "andi@example.com", creatorID, time.Time{}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := access.NewCandidate(tt.id, tt.person, tt.email, tt.createdBy, tt.now)
			if tt.wantErr {
				require.ErrorIs(t, err, access.ErrInvalidCandidate)
				require.Nil(t, got)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.id, got.SubjectID())
			snapshot := got.Snapshot()
			require.Equal(t, tt.person, snapshot.Name)
			require.Equal(t, tt.email, snapshot.Email)
			require.Equal(t, tt.createdBy, snapshot.CreatedBy)
			require.Equal(t, tt.now, snapshot.CreatedAt)
			require.Equal(t, tt.now, snapshot.UpdatedAt)
		})
	}
}

// candidateRestoreCase defines a malformed or historical candidate snapshot mutation.
type candidateRestoreCase struct {
	name    string
	alter   func(*access.CandidateSnapshot)
	wantErr bool
}

// TestRestoreCandidatePreservesContactHistory rejects corrupt identity while keeping stored contact timestamps.
func TestRestoreCandidatePreservesContactHistory(t *testing.T) {
	now := testdata.Now()
	created, err := access.NewCandidate(candidateID, "Andi Pratama", "andi@example.com", creatorID, now)
	require.NoError(t, err)
	tests := []candidateRestoreCase{
		{"updated later", func(s *access.CandidateSnapshot) { s.UpdatedAt = now.Add(time.Minute) }, false},
		{"empty id", func(s *access.CandidateSnapshot) { s.ID = "" }, true},
		{"empty name", func(s *access.CandidateSnapshot) { s.Name = "" }, true},
		{"empty email", func(s *access.CandidateSnapshot) { s.Email = "" }, true},
		{"empty creator", func(s *access.CandidateSnapshot) { s.CreatedBy = "" }, true},
		{"updated before created", func(s *access.CandidateSnapshot) { s.UpdatedAt = now.Add(-time.Millisecond) }, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			snapshot := created.Snapshot()
			tt.alter(&snapshot)
			got, err := access.RestoreCandidate(snapshot)
			if tt.wantErr {
				require.ErrorIs(t, err, access.ErrInvalidCandidate)
				require.Nil(t, got)
				return
			}
			require.NoError(t, err)
			require.Equal(t, snapshot.UpdatedAt, got.Snapshot().UpdatedAt)
			require.Equal(t, "andi@example.com", got.Snapshot().Email)
		})
	}
}
