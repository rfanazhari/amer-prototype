package errs

const (
	InvalidInput             = "INVALID_INPUT"
	NotFound                 = "NOT_FOUND"
	MethodNotAllowed         = "METHOD_NOT_ALLOWED"
	InvalidTransition        = "INVALID_TRANSITION"
	EventUnavailable         = "EVENT_UNAVAILABLE"
	InternalError            = "INTERNAL_ERROR"
	ShuttingDown             = "SHUTTING_DOWN"
	SessionUnavailable       = "SESSION_UNAVAILABLE"
	IAUnauthenticated        = "IA_UNAUTHENTICATED"
	IAForbidden              = "IA_FORBIDDEN"
	IAAccountConflict        = "IA_ACCOUNT_CONFLICT"
	IAInvitationUnavailable  = "IA_INVITATION_UNAVAILABLE"
	IARateLimited            = "IA_RATE_LIMITED"
	IADependencyUnavailable  = "IA_DEPENDENCY_UNAVAILABLE"
	IANoActiveRevision       = "IA_NO_ACTIVE_REVISION"
	IAUnsupportedTemplate    = "IA_UNSUPPORTED_TEMPLATE"
	ACPUnauthenticated       = "ACP_UNAUTHENTICATED"
	ACPForbidden             = "ACP_FORBIDDEN"
	ACPIdempotencyConflict   = "ACP_IDEMPOTENCY_CONFLICT"
	ACPDraftConflict         = "ACP_DRAFT_CONFLICT"
	ACPValidationConflict    = "ACP_VALIDATION_CONFLICT"
	ACPPublicationBlocked    = "ACP_PUBLICATION_BLOCKED"
	ACPDependencyUnavailable = "ACP_DEPENDENCY_UNAVAILABLE"
	ACPPublicationUncertain  = "ACP_PUBLICATION_UNCERTAIN"

	CINForbidden             = "CIN_FORBIDDEN"
	CINConsentRequired       = "CIN_CONSENT_REQUIRED"
	CINSessionCoverage       = "CIN_SESSION_COVERAGE"
	CINIdempotencyConflict   = "CIN_IDEMPOTENCY_CONFLICT"
	CINControllerConflict    = "CIN_CONTROLLER_CONFLICT"
	CINNotController         = "CIN_NOT_CONTROLLER"
	CINTerminal              = "CIN_TERMINAL"
	CINNotLive               = "CIN_NOT_LIVE"
	CINNoticeUnavailable     = "CIN_NOTICE_UNAVAILABLE"
	CINDependencyUnavailable = "CIN_DEPENDENCY_UNAVAILABLE"
	CINStartUncertain        = "CIN_START_UNCERTAIN"

	EXLNotFound              = "EXL_NOT_FOUND"
	EXLAssignmentUnavailable = "EXL_ASSIGNMENT_UNAVAILABLE"
	EXLIdempotencyConflict   = "EXL_IDEMPOTENCY_CONFLICT"
	EXLInvalidTransition     = "EXL_INVALID_TRANSITION"
	EXLStaleVersion          = "EXL_STALE_VERSION"
	EXLStaleAttempt          = "EXL_STALE_ATTEMPT"
	EXLProgressRequired      = "EXL_PROGRESS_REQUIRED"
	EXLTerminalConflict      = "EXL_TERMINAL_CONFLICT"
	EXLDependencyUnavailable = "EXL_DEPENDENCY_UNAVAILABLE"
	EXLUncertainCommit       = "EXL_UNCERTAIN_COMMIT"
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
