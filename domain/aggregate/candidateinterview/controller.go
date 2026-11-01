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
	s := ControllerSnapshot{AssignmentID: assignmentID, ExecutionID: executionID, ControllerID: id, Version: 1, UpdatedAt: now}
	if !s.consistent() {
		return nil, ErrInvalidController
	}
	return &InterviewController{state: s}, nil
}

// consistent reports whether every field of the stored controller is present and the version is at least 1.
func (s ControllerSnapshot) consistent() bool {
	return s.AssignmentID != "" && s.ExecutionID != "" && s.ControllerID.value != "" &&
		s.Version >= 1 && !s.UpdatedAt.IsZero()
}

// RestoreController rebuilds a controller from stored data.
func RestoreController(s ControllerSnapshot) (*InterviewController, error) {
	if !s.consistent() {
		return nil, ErrInvalidController
	}
	return &InterviewController{state: s}, nil
}

// Snapshot returns a detached copy of the controller.
func (c *InterviewController) Snapshot() ControllerSnapshot { return c.state }

// IsCurrent reports whether id is the current controlling tab.
func (c *InterviewController) IsCurrent(id ControllerID) bool {
	return c.state.ControllerID == id
}

// Takeover replaces the controlling tab when expectedVersion matches and the execution is live.
func (c *InterviewController) Takeover(expectedVersion int64, next ControllerID, executionTerminal bool, now time.Time) error {
	if next.value == "" {
		return ErrInvalidControllerID
	}
	if executionTerminal {
		return ErrExecutionTerminal
	}
	if c.state.Version != expectedVersion {
		return ErrStaleControllerVersion
	}
	c.state.ControllerID = next
	c.state.Version++
	c.state.UpdatedAt = now
	return nil
}
