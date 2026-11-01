package accessport

import "errors"

var (
	// ErrNotFound reports that an administrative account or session was not stored.
	ErrNotFound = errors.New("administrative access record not found")
)

// AccountConflictError classifies a duplicate canonical administrative email without exposing existing identifiers.
type AccountConflictError struct{}

// Error returns a safe conflict description without email, role or identity details.
func (AccountConflictError) Error() string { return "administrative account conflict" }

// TokenConflictError classifies a duplicate administrative session digest.
type TokenConflictError struct{}

// Error returns a safe token-collision description without the digest value.
func (TokenConflictError) Error() string { return "administrative session token conflict" }
