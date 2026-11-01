package access

import "errors"

var (
	// ErrInvalidAccount reports rejected administrative account construction or restoration.
	ErrInvalidAccount = errors.New("invalid administrative account")
	// ErrInvalidSession reports rejected administrative session construction, restoration or revocation.
	ErrInvalidSession = errors.New("invalid administrative session")
)
