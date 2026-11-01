package service

import "github.com/rfanazhari/amer-prototype/pkg/errs"

// ExecutionError is an execution lifecycle failure that carries a public error code.
type ExecutionError struct {
	// Code is the public pkg/errs code that delivery and siblings map to a message.
	Code string
}

// Error returns the public code so logs and wrapped errors stay greppable.
func (e *ExecutionError) Error() string { return e.Code }

// Is matches another execution error with the same public code.
func (e *ExecutionError) Is(target error) bool {
	other, ok := target.(*ExecutionError)
	return ok && other.Code == e.Code
}

var (
	// ErrExecutionInvalidInput reports an illegal key, actor, scope, identifier or input-reference shape.
	ErrExecutionInvalidInput = &ExecutionError{Code: errs.InvalidInput}
	// ErrExecutionNotFound reports that the assignment or execution is absent.
	ErrExecutionNotFound = &ExecutionError{Code: errs.EXLNotFound}
	// ErrExecutionAssignmentUnavailable reports a revoked or already completed assignment.
	ErrExecutionAssignmentUnavailable = &ExecutionError{Code: errs.EXLAssignmentUnavailable}
	// ErrExecutionIdempotencyConflict reports a start identity reused with a different intent.
	ErrExecutionIdempotencyConflict = &ExecutionError{Code: errs.EXLIdempotencyConflict}
	// ErrExecutionInvalidTransition reports a source state that does not allow the action.
	ErrExecutionInvalidTransition = &ExecutionError{Code: errs.EXLInvalidTransition}
	// ErrExecutionStaleVersion reports an expected state version that does not match.
	ErrExecutionStaleVersion = &ExecutionError{Code: errs.EXLStaleVersion}
	// ErrExecutionStaleAttempt reports a stale attempt generation or a deadline that is not strictly ahead.
	ErrExecutionStaleAttempt = &ExecutionError{Code: errs.EXLStaleAttempt}
	// ErrExecutionProgressRequired reports that the progress port did not confirm committed progress.
	ErrExecutionProgressRequired = &ExecutionError{Code: errs.EXLProgressRequired}
	// ErrExecutionTerminalConflict reports that a different terminal outcome is already committed.
	ErrExecutionTerminalConflict = &ExecutionError{Code: errs.EXLTerminalConflict}
	// ErrExecutionDependencyUnavailable reports an unreadable pinned revision or progress store.
	ErrExecutionDependencyUnavailable = &ExecutionError{Code: errs.EXLDependencyUnavailable}
	// ErrExecutionUncertainCommit reports an unknown commit result; the same key may be retried.
	ErrExecutionUncertainCommit = &ExecutionError{Code: errs.EXLUncertainCommit}
)
