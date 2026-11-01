package execution

import "errors"

var (
	// ErrInvalidExecution reports rejected construction, restoration, or field validation for an execution.
	ErrInvalidExecution = errors.New("invalid execution state")
	// ErrInvalidTransition reports that the current state does not allow the requested transition.
	ErrInvalidTransition = errors.New("execution state does not allow this transition")
	// ErrStaleVersion reports that the expected state version does not match the stored version.
	ErrStaleVersion = errors.New("stale execution state version")
	// ErrStaleAttempt reports that the attempt generation is older than current authority or the deadline is not strictly ahead.
	ErrStaleAttempt = errors.New("stale attempt generation or deadline has passed")
	// ErrProgressRequired reports that completion was requested without a committed progress reference.
	ErrProgressRequired = errors.New("completion requires a committed progress reference")
	// ErrTerminalConflict reports that a different terminal outcome is already committed for this execution.
	ErrTerminalConflict = errors.New("a different terminal outcome is already committed")
)
