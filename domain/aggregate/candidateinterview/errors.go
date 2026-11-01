package candidateinterview

import "errors"

var (
	// ErrInvalidNoticeVersion reports a notice version that breaks the identifier character rule.
	ErrInvalidNoticeVersion = errors.New("invalid notice version")
	// ErrInvalidControllerID reports a controller id that breaks the identifier character rule.
	ErrInvalidControllerID = errors.New("invalid controller id")
	// ErrInvalidConsent reports rejected consent construction or restoration.
	ErrInvalidConsent = errors.New("invalid consent")
	// ErrNoticeNotCurrent reports an acceptance of a notice version other than the current one.
	ErrNoticeNotCurrent = errors.New("notice version is not the current version")
	// ErrInvalidController reports rejected controller construction or restoration.
	ErrInvalidController = errors.New("invalid interview controller")
	// ErrStaleControllerVersion reports that the expected controller version does not match the stored version.
	ErrStaleControllerVersion = errors.New("stale interview controller version")
	// ErrExecutionTerminal reports that a takeover was requested for an execution that has ended.
	ErrExecutionTerminal = errors.New("execution is terminal")
)
