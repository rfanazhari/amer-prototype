package errs

const (
	InvalidInput      = "INVALID_INPUT"
	NotFound          = "NOT_FOUND"
	MethodNotAllowed  = "METHOD_NOT_ALLOWED"
	InvalidTransition = "INVALID_TRANSITION"
	EventUnavailable  = "EVENT_UNAVAILABLE"
	InternalError     = "INTERNAL_ERROR"
	ShuttingDown      = "SHUTTING_DOWN"
)

// Technical preserves diagnostic causes while preventing business error mapping.
type Technical struct{ Cause error }

func (e Technical) Error() string {
	if e.Cause == nil {
		return "technical failure"
	}
	return "technical failure: " + e.Cause.Error()
}
func (e Technical) Unwrap() error { return e.Cause }
