package valueobject

import (
	"errors"
	"time"
)

// ErrInvalidSessionLifetime reports rejected lifetime construction or restoration.
var ErrInvalidSessionLifetime = errors.New("invalid session lifetime")

// AdministrativeDuration is the exact created-to-expiry span for administrative and candidate sessions.
const AdministrativeDuration = 2 * time.Hour

// SessionLifetime is an immutable created/expiry pair with exact-boundary current access semantics.
type SessionLifetime struct {
	createdAt time.Time
	expiresAt time.Time
}

// NewSessionLifetime records expiry as createdAt plus duration and requires expiry after creation.
func NewSessionLifetime(createdAt time.Time, duration time.Duration) (SessionLifetime, error) {
	return SessionLifetime{}, ErrInvalidSessionLifetime
}

// RestoreSessionLifetime accepts stored created and expiry instants without recalculating the deadline.
func RestoreSessionLifetime(createdAt, expiresAt time.Time) (SessionLifetime, error) {
	return SessionLifetime{}, ErrInvalidSessionLifetime
}

// CreatedAt returns the trusted creation instant.
func (l SessionLifetime) CreatedAt() time.Time { return time.Time{} }

// ExpiresAt returns the stored absolute expiry instant.
func (l SessionLifetime) ExpiresAt() time.Time { return time.Time{} }

// Current reports whether now is strictly before expiry; equality is expired.
func (l SessionLifetime) Current(now time.Time) bool { return false }

// Valid reports whether this value was initialized with ordered nonzero instants; the Go zero value is invalid.
func (l SessionLifetime) Valid() bool { return false }
