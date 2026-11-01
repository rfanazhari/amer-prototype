package service

import "errors"

var (
	// ErrDraftConflict reports a stale expected draft version, including an edit race during validation.
	ErrDraftConflict = errors.New("agent draft version conflicts with the current draft")
	// ErrIdempotencyConflict reports a stored publication key whose intent does not match the request.
	ErrIdempotencyConflict = errors.New("publication command identity is already used with a different intent")
	// ErrValidationConflict reports an owned validation that does not match the current draft interpretation.
	ErrValidationConflict = errors.New("validation does not match the current draft")
	// ErrPublicationBlocked reports an eligible-draft check that found blocking validation errors.
	ErrPublicationBlocked = errors.New("publication is blocked by validation errors")
	// ErrDependencyUnavailable reports a reference or store outage before durable acceptance.
	ErrDependencyUnavailable = errors.New("agent configuration dependency is unavailable")
)
