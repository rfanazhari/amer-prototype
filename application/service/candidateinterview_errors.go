package service

// InterviewErrorKind classifies a candidate-interview failure so delivery can map it without string matching.
type InterviewErrorKind string

const (
	// InterviewNotFound marks an absent consent, controller or execution inside the caller's assignment.
	InterviewNotFound InterviewErrorKind = "not_found"
	// InterviewConflict marks a stale version, a reused idempotency intent or a lost takeover race.
	InterviewConflict InterviewErrorKind = "conflict"
	// InterviewCoverage marks a candidate session that does not outlast the interview budget.
	InterviewCoverage InterviewErrorKind = "coverage"
	// InterviewDependencyUnavailable marks a missing or failing sibling port or store.
	InterviewDependencyUnavailable InterviewErrorKind = "dependency_unavailable"
)

// InterviewError is a classified candidate-interview failure.
type InterviewError struct {
	// Kind is the classification delivery maps to a public code.
	Kind InterviewErrorKind
}

// Error returns the kind only, so logs and wrapped errors never carry request data.
func (e *InterviewError) Error() string { return "" }

// Is reports whether target is a candidate-interview error of the same kind.
func (e *InterviewError) Is(target error) bool { return false }

var (
	// ErrInterviewNotFound reports an absent consent, controller or execution.
	ErrInterviewNotFound = &InterviewError{}
	// ErrInterviewConflict reports a stale version, a reused idempotency intent or a lost takeover race.
	ErrInterviewConflict = &InterviewError{}
	// ErrInterviewCoverage reports a session that does not cover the interview budget.
	ErrInterviewCoverage = &InterviewError{}
	// ErrInterviewDependencyUnavailable reports an unbound or failing sibling port.
	ErrInterviewDependencyUnavailable = &InterviewError{}
)

// InterviewErrorKindOf returns the kind of the first candidate-interview error in err's chain.
func InterviewErrorKindOf(err error) (InterviewErrorKind, bool) { return "", false }
