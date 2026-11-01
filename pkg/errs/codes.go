package errs

const (
	InvalidInput            = "INVALID_INPUT"
	NotFound                = "NOT_FOUND"
	MethodNotAllowed        = "METHOD_NOT_ALLOWED"
	InvalidTransition       = "INVALID_TRANSITION"
	EventUnavailable        = "EVENT_UNAVAILABLE"
	InternalError           = "INTERNAL_ERROR"
	ShuttingDown            = "SHUTTING_DOWN"
	SessionUnavailable      = "SESSION_UNAVAILABLE"
	IAUnauthenticated       = "IA_UNAUTHENTICATED"
	IAForbidden             = "IA_FORBIDDEN"
	IAAccountConflict       = "IA_ACCOUNT_CONFLICT"
	IAInvitationUnavailable = "IA_INVITATION_UNAVAILABLE"
	IARateLimited           = "IA_RATE_LIMITED"
	IADependencyUnavailable = "IA_DEPENDENCY_UNAVAILABLE"
)

// Technical preserves diagnostic causes while preventing business error mapping.
type Technical struct{ Cause error }

// Error returns a diagnostic description suitable for internal logging, including a cause when present.
func (e Technical) Error() string {
	if e.Cause == nil {
		return "technical failure"
	}
	return "technical failure: " + e.Cause.Error()
}

// Unwrap preserves the underlying cause for internal error inspection.
func (e Technical) Unwrap() error { return e.Cause }
