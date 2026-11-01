package accessport

import "context"

// CommitUncertainError means login or logout persistence could not be confirmed.
type CommitUncertainError struct {
	Cause error
}

// Error returns a safe diagnostic without including the underlying storage error.
func (e *CommitUncertainError) Error() string { return "pending" }

// Unwrap preserves the technical cause for internal classification.
func (e *CommitUncertainError) Unwrap() error { return e.Cause }

// LoginTx exposes locked-account recheck and session insert inside one transaction.
type LoginTx interface {
	// Account returns the locked account for eligibility and stored-hash recheck before insert.
	Account(ctx context.Context) (AccountRecord, error)
	// InsertSession stores the session in the open transaction or returns TokenConflictError.
	InsertSession(ctx context.Context, record SessionRecord) error
}

// SessionIssuance serializes administrative login writes after out-of-transaction password proof.
type SessionIssuance interface {
	// WithinLogin locks the account, runs the callback, and rolls back on callback or known failure.
	// A successful callback is not a durable session until commit succeeds.
	WithinLogin(ctx context.Context, accountID string, fn func(LoginTx) error) error
}
