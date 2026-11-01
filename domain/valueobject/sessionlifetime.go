package valueobject

import (
	"errors"
	"time"
)

// ErrInvalidSessionLifetime reports rejected lifetime construction or restoration.
var ErrInvalidSessionLifetime = errors.New("invalid session lifetime")

// AdministrativeDuration is the exact created-to-expiry span for administrative and candidate sessions.
const AdministrativeDuration = 2 * time.Hour

// InvitationDuration is the exact created-to-expiry span for assignment invitations.
const InvitationDuration = 24 * time.Hour

// SessionLifetime is an immutable created/expiry pair with exact-boundary current access semantics.
type SessionLifetime struct {
	createdAt time.Time
	expiresAt time.Time
}

// NewSessionLifetime records expiry as createdAt plus duration and requires expiry after creation.
func NewSessionLifetime(createdAt time.Time, duration time.Duration) (SessionLifetime, error) {
	if createdAt.IsZero() || duration <= 0 {
		return SessionLifetime{}, ErrInvalidSessionLifetime
	}
	expiresAt := createdAt.Add(duration)
	if !expiresAt.After(createdAt) {
		return SessionLifetime{}, ErrInvalidSessionLifetime
	}
	return SessionLifetime{createdAt: createdAt, expiresAt: expiresAt}, nil
}

// RestoreSessionLifetime accepts stored created and expiry instants without recalculating the deadline.
func RestoreSessionLifetime(createdAt, expiresAt time.Time) (SessionLifetime, error) {
	if createdAt.IsZero() || expiresAt.IsZero() || !expiresAt.After(createdAt) {
		return SessionLifetime{}, ErrInvalidSessionLifetime
	}
	return SessionLifetime{createdAt: createdAt, expiresAt: expiresAt}, nil
}

// CreatedAt returns the trusted creation instant.
func (l SessionLifetime) CreatedAt() time.Time { return l.createdAt }

// ExpiresAt returns the stored absolute expiry instant.
func (l SessionLifetime) ExpiresAt() time.Time { return l.expiresAt }

// Current reports whether now is strictly before expiry; equality is expired.
func (l SessionLifetime) Current(now time.Time) bool {
	return l.Valid() && now.Before(l.expiresAt)
}

// Valid reports whether this value was initialized with ordered nonzero instants; the Go zero value is invalid.
func (l SessionLifetime) Valid() bool {
	return !l.createdAt.IsZero() && l.expiresAt.After(l.createdAt)
}
