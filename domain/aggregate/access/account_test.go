package access_test

import (
	"testing"
	"time"

	"github.com/rfanazhari/amer-prototype/domain/aggregate/access"
	"github.com/rfanazhari/amer-prototype/domain/enum"
	"github.com/rfanazhari/amer-prototype/test/testdata"
	"github.com/stretchr/testify/require"
)

const accountID = "3f1a2c4e-8b90-4d11-9c22-aa11bb22cc33"
const otherAccountID = "7a8b9c0d-1e2f-3456-7890-abcdef123456"

// accountCreateCase defines identity and role inputs for a new administrative account.
type accountCreateCase struct {
	name    string
	id      string
	role    enum.Role
	now     time.Time
	wantErr bool
}

// TestNewAdministrativeAccountCreatesActiveFixedRoleIdentity records stable subject identity without deriving it from a session.
func TestNewAdministrativeAccountCreatesActiveFixedRoleIdentity(t *testing.T) {
	now := testdata.Now()
	tests := []accountCreateCase{
		{"admin", accountID, enum.Admin, now, false},
		{"hr", accountID, enum.HR, now, false},
		{"empty id", "", enum.Admin, now, true},
		{"invalid role", accountID, enum.Role("CANDIDATE"), now, true},
		{"zero time", accountID, enum.Admin, time.Time{}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := access.NewAdministrativeAccount(tt.id, tt.role, tt.now)
			if tt.wantErr {
				require.ErrorIs(t, err, access.ErrInvalidAccount)
				require.Nil(t, got)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.id, got.SubjectID())
			require.True(t, got.Eligible())
			snapshot := got.Snapshot()
			require.Equal(t, tt.id, snapshot.ID)
			require.Equal(t, tt.role, snapshot.Role)
			require.Equal(t, enum.Active, snapshot.Status)
			require.Equal(t, tt.now, snapshot.CreatedAt)
			require.Equal(t, tt.now, snapshot.UpdatedAt)
			session, err := access.NewAdministrativeSession("sess-1", otherAccountID, tt.now)
			require.NoError(t, err)
			require.NotEqual(t, got.SubjectID(), session.Snapshot().ID)
			require.Equal(t, got.SubjectID(), snapshot.ID)
		})
	}
}

// accountRestoreCase defines a malformed or historical account snapshot mutation.
type accountRestoreCase struct {
	name    string
	alter   func(*access.AccountSnapshot)
	wantErr bool
}

// TestRestoreAdministrativeAccountPreservesHistory rejects corrupt identity while keeping stored role, status and timestamps.
func TestRestoreAdministrativeAccountPreservesHistory(t *testing.T) {
	now := testdata.Now()
	created, err := access.NewAdministrativeAccount(accountID, enum.HR, now)
	require.NoError(t, err)
	tests := []accountRestoreCase{
		{"disabled remains disabled", func(s *access.AccountSnapshot) {
			s.Status = enum.Disabled
			s.UpdatedAt = now.Add(time.Minute)
		}, false},
		{"empty id", func(s *access.AccountSnapshot) { s.ID = "" }, true},
		{"invalid role", func(s *access.AccountSnapshot) { s.Role = enum.Role("SUPER") }, true},
		{"invalid status", func(s *access.AccountSnapshot) { s.Status = enum.AccountStatus("PENDING") }, true},
		{"missing created", func(s *access.AccountSnapshot) { s.CreatedAt = time.Time{} }, true},
		{"updated before created", func(s *access.AccountSnapshot) { s.UpdatedAt = now.Add(-time.Millisecond) }, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			snapshot := created.Snapshot()
			tt.alter(&snapshot)
			got, err := access.RestoreAdministrativeAccount(snapshot)
			if tt.wantErr {
				require.ErrorIs(t, err, access.ErrInvalidAccount)
				return
			}
			require.NoError(t, err)
			require.Equal(t, snapshot, got.Snapshot())
			require.False(t, got.Eligible())
			require.Equal(t, accountID, got.SubjectID())
			require.Equal(t, now, got.Snapshot().CreatedAt)
			require.Equal(t, now.Add(time.Minute), got.Snapshot().UpdatedAt)
		})
	}
}
