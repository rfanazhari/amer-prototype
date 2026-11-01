package access_test

import (
	"testing"
	"time"

	"github.com/rfanazhari/amer-prototype/domain/aggregate/access"
	"github.com/rfanazhari/amer-prototype/domain/enum"
	"github.com/rfanazhari/amer-prototype/domain/valueobject"
	"github.com/rfanazhari/amer-prototype/test/testdata"
	"github.com/stretchr/testify/require"
)

const sessionID = "c0ffee00-1111-2222-3333-444455556666"

// sessionCreateCase defines identity inputs for a new administrative session.
type sessionCreateCase struct {
	name      string
	id        string
	accountID string
	now       time.Time
	wantErr   bool
}

// TestNewAdministrativeSessionSetsExactTwoHourExpiry records independent session identity and absolute expiry.
func TestNewAdministrativeSessionSetsExactTwoHourExpiry(t *testing.T) {
	now := testdata.Now()
	tests := []sessionCreateCase{
		{"valid", sessionID, accountID, now, false},
		{"empty id", "", accountID, now, true},
		{"empty account", sessionID, "", now, true},
		{"zero time", sessionID, accountID, time.Time{}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := access.NewAdministrativeSession(tt.id, tt.accountID, tt.now)
			if tt.wantErr {
				require.ErrorIs(t, err, access.ErrInvalidSession)
				require.Nil(t, got)
				return
			}
			require.NoError(t, err)
			snapshot := got.Snapshot()
			require.Equal(t, tt.id, snapshot.ID)
			require.Equal(t, tt.accountID, snapshot.AccountID)
			require.Equal(t, tt.now, snapshot.CreatedAt)
			require.Equal(t, tt.now.Add(valueobject.AdministrativeDuration), snapshot.ExpiresAt)
			require.Nil(t, snapshot.RevokedAt)
			require.True(t, got.Current(tt.now))
			require.False(t, got.Current(tt.now.Add(valueobject.AdministrativeDuration)))
		})
	}
}

// sessionCurrentCase defines exact expiry observations for an unrevoked session.
type sessionCurrentCase struct {
	name    string
	offset  time.Duration
	current bool
}

// TestAdministrativeSessionCurrentUsesExactExpiryBoundary denies access at the stored expiry instant.
func TestAdministrativeSessionCurrentUsesExactExpiryBoundary(t *testing.T) {
	now := testdata.Now()
	session, err := access.NewAdministrativeSession(sessionID, accountID, now)
	require.NoError(t, err)
	tests := []sessionCurrentCase{
		{"before expiry", valueobject.AdministrativeDuration - time.Millisecond, true},
		{"at expiry", valueobject.AdministrativeDuration, false},
		{"after expiry", valueobject.AdministrativeDuration + time.Millisecond, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.current, session.Current(now.Add(tt.offset)))
		})
	}
}

// TestAdministrativeSessionRevocationCopiesPointersAndDeniesAccess proves logout revocation and alias isolation.
func TestAdministrativeSessionRevocationCopiesPointersAndDeniesAccess(t *testing.T) {
	now := testdata.Now()
	session, err := access.NewAdministrativeSession(sessionID, accountID, now)
	require.NoError(t, err)
	revokedAt := now.Add(time.Minute)
	require.NoError(t, session.Revoke(revokedAt))
	require.False(t, session.Current(revokedAt))
	snapshot := session.Snapshot()
	require.Equal(t, revokedAt, *snapshot.RevokedAt)
	*snapshot.RevokedAt = now.Add(time.Hour)
	require.Equal(t, revokedAt, *session.Snapshot().RevokedAt)
	require.ErrorIs(t, session.Revoke(revokedAt.Add(time.Minute)), access.ErrInvalidSession)
	zero := new(access.AdministrativeSession)
	require.ErrorIs(t, zero.Revoke(now), access.ErrInvalidSession)
}

// sessionRestoreCase defines stored session mutations expected to restore or fail.
type sessionRestoreCase struct {
	name    string
	alter   func(*access.SessionSnapshot)
	wantErr bool
}

// TestRestoreAdministrativeSessionPreservesDeadlineAndRevocation restores history without resetting expiry.
func TestRestoreAdministrativeSessionPreservesDeadlineAndRevocation(t *testing.T) {
	now := testdata.Now()
	created, err := access.NewAdministrativeSession(sessionID, accountID, now)
	require.NoError(t, err)
	revoked := now.Add(time.Minute)
	tests := []sessionRestoreCase{
		{"historical expiry preserved", func(s *access.SessionSnapshot) {
			s.ExpiresAt = now.Add(90 * time.Minute)
		}, false},
		{"revoked pointer copied", func(s *access.SessionSnapshot) { s.RevokedAt = &revoked }, false},
		{"empty id", func(s *access.SessionSnapshot) { s.ID = "" }, true},
		{"empty account", func(s *access.SessionSnapshot) { s.AccountID = "" }, true},
		{"missing created", func(s *access.SessionSnapshot) { s.CreatedAt = time.Time{} }, true},
		{"expiry equal created", func(s *access.SessionSnapshot) { s.ExpiresAt = s.CreatedAt }, true},
		{"revoked before created", func(s *access.SessionSnapshot) {
			past := now.Add(-time.Millisecond)
			s.RevokedAt = &past
		}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			snapshot := created.Snapshot()
			tt.alter(&snapshot)
			got, err := access.RestoreAdministrativeSession(snapshot)
			if tt.wantErr {
				require.ErrorIs(t, err, access.ErrInvalidSession)
				return
			}
			require.NoError(t, err)
			if snapshot.RevokedAt != nil {
				owned := *got.Snapshot().RevokedAt
				*snapshot.RevokedAt = now.Add(3 * time.Hour)
				require.Equal(t, owned, *got.Snapshot().RevokedAt)
				detached := got.Snapshot()
				*detached.RevokedAt = now.Add(4 * time.Hour)
				require.Equal(t, owned, *got.Snapshot().RevokedAt)
				require.False(t, got.Current(now.Add(time.Minute)))
				return
			}
			require.Equal(t, now.Add(90*time.Minute), got.Snapshot().ExpiresAt)
			require.True(t, got.Current(now.Add(90*time.Minute-time.Millisecond)))
			require.False(t, got.Current(now.Add(90*time.Minute)))
		})
	}
}

// authorizeCase defines current-account eligibility for session authorization.
type authorizeCase struct {
	name      string
	role      enum.Role
	status    enum.AccountStatus
	accountID string
	expired   bool
	revoked   bool
	want      bool
}

// TestAdministrativeSessionAuthorizesOnlyEligibleBoundAccounts requires a current session and matching ACTIVE account.
func TestAdministrativeSessionAuthorizesOnlyEligibleBoundAccounts(t *testing.T) {
	now := testdata.Now()
	tests := []authorizeCase{
		{"active matching", enum.HR, enum.Active, accountID, false, false, true},
		{"disabled matching", enum.HR, enum.Disabled, accountID, false, false, false},
		{"other account", enum.HR, enum.Active, otherAccountID, false, false, false},
		{"expired", enum.Admin, enum.Active, accountID, true, false, false},
		{"revoked", enum.Admin, enum.Active, accountID, false, true, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			session, err := access.NewAdministrativeSession(sessionID, accountID, now)
			require.NoError(t, err)
			if tt.revoked {
				require.NoError(t, session.Revoke(now.Add(time.Minute)))
			}
			account, err := access.NewAdministrativeAccount(tt.accountID, tt.role, now)
			require.NoError(t, err)
			if tt.status == enum.Disabled {
				snapshot := account.Snapshot()
				snapshot.Status = enum.Disabled
				account, err = access.RestoreAdministrativeAccount(snapshot)
				require.NoError(t, err)
			}
			when := now.Add(time.Minute)
			if tt.expired {
				when = now.Add(valueobject.AdministrativeDuration)
			}
			require.Equal(t, tt.want, session.Authorizes(account, when))
			require.False(t, session.Authorizes(nil, when))
		})
	}
}
