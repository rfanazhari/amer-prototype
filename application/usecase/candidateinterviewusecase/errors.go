package candidateinterviewusecase

import "errors"

var (
	// ErrConsentRequired reports a start without a stored acceptance of the current notice version.
	ErrConsentRequired = errors.New("acceptance of the current notice is required")
	// ErrNotController reports a command from a tab that is not the current controller.
	ErrNotController = errors.New("caller is not the interview controller")
	// ErrInvalidIdempotencyKey reports an Idempotency-Key that breaks the local character rule.
	ErrInvalidIdempotencyKey = errors.New("invalid idempotency key")
)

// errNotImplemented marks an operation whose behavior has not been written yet.
var errNotImplemented = errors.New("not implemented")
