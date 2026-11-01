package candidateinterview

import "time"

// ControllerSnapshot is the stored shape of the controlling tab.
type ControllerSnapshot struct {
	// AssignmentID is the assignment the controller belongs to.
	AssignmentID string
	// ExecutionID is the execution being controlled.
	ExecutionID string
	// ControllerID is the current controlling tab.
	ControllerID ControllerID
	// Version is the takeover counter, starting at 1.
	Version int64
	// UpdatedAt is the trusted instant of the last claim or takeover.
	UpdatedAt time.Time
}

// InterviewController is the aggregate root for the single controlling tab of an assignment.
type InterviewController struct{ state ControllerSnapshot }

// ClaimController creates the first controller at version 1.
func ClaimController(assignmentID, executionID string, id ControllerID, now time.Time) (*InterviewController, error) {
	return nil, ErrInvalidController
}

// RestoreController rebuilds a controller from stored data.
func RestoreController(s ControllerSnapshot) (*InterviewController, error) {
	return nil, ErrInvalidController
}

// Snapshot returns a detached copy of the controller.
func (c *InterviewController) Snapshot() ControllerSnapshot { return ControllerSnapshot{} }

// IsCurrent reports whether id is the current controlling tab.
func (c *InterviewController) IsCurrent(id ControllerID) bool { return false }

// Takeover replaces the controlling tab when expectedVersion matches and the execution is live.
func (c *InterviewController) Takeover(expectedVersion int64, next ControllerID, executionTerminal bool, now time.Time) error {
	return ErrInvalidController
}
