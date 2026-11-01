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
func copyTime(t *time.Time) *time.Time {
	if t == nil {
		return nil
	}
	v := *t
	return &v
}

// NewAdministrativeSession records a new unrevoked session whose expiry is exactly creation plus two hours.
func NewAdministrativeSession(id, accountID string, now time.Time) (*AdministrativeSession, error) {
	if id == "" || accountID == "" {
		return nil, ErrInvalidSession
	}
	lifetime, err := valueobject.NewSessionLifetime(now, valueobject.AdministrativeDuration)
	if err != nil {
		return nil, ErrInvalidSession
	}
	return &AdministrativeSession{id: id, accountID: accountID, lifetime: lifetime}, nil
}

// RestoreAdministrativeSession validates stored identity, account reference and time order without resetting the deadline.
func RestoreAdministrativeSession(s SessionSnapshot) (*AdministrativeSession, error) {
	if s.ID == "" || s.AccountID == "" {
		return nil, ErrInvalidSession
	}
	lifetime, err := valueobject.RestoreSessionLifetime(s.CreatedAt, s.ExpiresAt)
	if err != nil {
		return nil, ErrInvalidSession
	}
	if s.RevokedAt != nil && (s.RevokedAt.IsZero() || s.RevokedAt.Before(s.CreatedAt)) {
		return nil, ErrInvalidSession
	}
	return &AdministrativeSession{
		id:        s.ID,
		accountID: s.AccountID,
		lifetime:  lifetime,
		revokedAt: copyTime(s.RevokedAt),
	}, nil
}

// Snapshot returns a detached copy of the current session facts, including an optional revocation pointer.
func (s *AdministrativeSession) Snapshot() SessionSnapshot {
	if s == nil {
		return SessionSnapshot{}
	}
	return SessionSnapshot{
		ID:        s.id,
		AccountID: s.accountID,
		CreatedAt: s.lifetime.CreatedAt(),
		ExpiresAt: s.lifetime.ExpiresAt(),
		RevokedAt: copyTime(s.revokedAt),
	}
}

// Current reports whether the session is unrevoked and now is strictly before expiry.
func (s *AdministrativeSession) Current(now time.Time) bool {
	return s != nil && s.id != "" && s.revokedAt == nil && s.lifetime.Current(now)
}

// Revoke records trusted revocation time on a valid unrevoked session.
func (s *AdministrativeSession) Revoke(now time.Time) error {
	if s == nil || s.id == "" || s.accountID == "" || !s.lifetime.Valid() || s.revokedAt != nil || now.IsZero() || now.Before(s.lifetime.CreatedAt()) {
		return ErrInvalidSession
	}
	s.revokedAt = copyTime(&now)
	return nil
}

// Authorizes reports whether this current session is bound to an eligible matching account.
func (s *AdministrativeSession) Authorizes(account *AdministrativeAccount, now time.Time) bool {
	return s.Current(now) && account != nil && account.Eligible() && account.SubjectID() == s.accountID
}
