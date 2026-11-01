package dto

import "time"

// NoticeView is the notice a candidate is asked to accept.
type NoticeView struct {
	// Version is the notice version.
	Version string
	// Statement is the approved notice text.
	Statement string
}

// ConsentView is the acceptance of the current notice version.
type ConsentView struct {
	// ID is the consent identity.
	ID string
	// NoticeVersion is the accepted notice version.
	NoticeVersion string
	// AcceptedAt is the UTC millisecond acceptance instant.
	AcceptedAt time.Time
}

// InterviewExecutionView is the candidate-safe execution; it has no controller id, report or salary field.
type InterviewExecutionView struct {
	// ID is the execution identity.
	ID string
	// State is CREATED, RUNNING, COMPLETED, FAILED or CANCELLED.
	State string
	// Version is the execution state version.
	Version int64
	// HardDeadline is nil until the execution is RUNNING.
	HardDeadline *time.Time
	// IsCurrentController is true when the calling tab is the controller.
	IsCurrentController bool
	// ControllerVersion is the takeover counter.
	ControllerVersion int64
}

// ProgressView is the progress summary runtime marks as visible to the candidate.
type ProgressView struct {
	// Phase is the current phase label.
	Phase string
	// Connection is connected, reconnecting or finalizing.
	Connection string
}

// InterviewView is the screen model for one candidate session.
type InterviewView struct {
	// Notice is nil when the notice catalog is empty.
	Notice *NoticeView
	// Consent is nil until the current notice version is accepted.
	Consent *ConsentView
	// Execution is nil until an execution exists for the assignment.
	Execution *InterviewExecutionView
	// Actions is a subset of accept, start, takeover and cancel.
	Actions []string
	// Progress is nil when runtime supplies none.
	Progress *ProgressView
}

// TakeoverView is the result of a controller takeover.
type TakeoverView struct {
	// IsCurrentController is true after a successful takeover.
	IsCurrentController bool
	// ControllerVersion is the new takeover counter.
	ControllerVersion int64
}
