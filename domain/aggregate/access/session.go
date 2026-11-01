package access

import (
	"time"

	"github.com/rfanazhari/amer-prototype/domain/valueobject"
)

// SessionSnapshot transfers persisted administrative session facts without exposing the live aggregate. Pointer fields are copied at creation, restoration and observation boundaries.
type SessionSnapshot struct {
	ID        string
	AccountID string
	CreatedAt time.Time
	ExpiresAt time.Time
	RevokedAt *time.Time
}

// AdministrativeSession owns independently identified administrative access lifetime and revocation.
type AdministrativeSession struct {
	id        string
	accountID string
	lifetime  valueobject.SessionLifetime
	revokedAt *time.Time
}

// copyTime returns an independent optional instant so callers cannot mutate session-owned dates through aliases.
func copyTime(t *time.Time) *time.Time { return nil }

// NewAdministrativeSession records a new unrevoked session whose expiry is exactly creation plus two hours.
func NewAdministrativeSession(id, accountID string, now time.Time) (*AdministrativeSession, error) {
	return nil, ErrInvalidSession
}

// RestoreAdministrativeSession validates stored identity, account reference and time order without resetting the deadline.
func RestoreAdministrativeSession(s SessionSnapshot) (*AdministrativeSession, error) {
	return nil, ErrInvalidSession
}

// Snapshot returns a detached copy of the current session facts, including an optional revocation pointer.
func (s *AdministrativeSession) Snapshot() SessionSnapshot { return SessionSnapshot{} }

// Current reports whether the session is unrevoked and now is strictly before expiry.
func (s *AdministrativeSession) Current(now time.Time) bool { return false }

// Revoke records trusted revocation time on a valid unrevoked session.
func (s *AdministrativeSession) Revoke(now time.Time) error { return ErrInvalidSession }

// Authorizes reports whether this current session is bound to an eligible matching account.
func (s *AdministrativeSession) Authorizes(account *AdministrativeAccount, now time.Time) bool {
	return false
}
