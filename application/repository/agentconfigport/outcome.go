// Package agentconfigport defines application persistence boundaries for agent configuration.
package agentconfigport

// CommitUncertainError means acceptance could not be determined; callers must
// reconcile the original command identity instead of assuming rollback.
type CommitUncertainError struct {
	Cause error
}

// Error returns a safe diagnostic without including the underlying storage error.
func (e *CommitUncertainError) Error() string { return "publication commit outcome uncertain" }

// Unwrap preserves the technical cause for internal classification.
func (e *CommitUncertainError) Unwrap() error { return nil }
