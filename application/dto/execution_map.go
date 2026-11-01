package dto

import "github.com/rfanazhari/amer-prototype/domain/aggregate/execution"

// NewExecutionView copies a committed snapshot into a detached view.
// assignmentID is empty for an execution that has no interview pin.
func NewExecutionView(snap execution.Snapshot, assignmentID string) *ExecutionView {
	view := &ExecutionView{
		ID: snap.ID, AssignmentID: assignmentID,
		InstanceID: snap.Binding.InstanceID, RevisionID: snap.Binding.RevisionID,
		State: string(snap.State), StateVersion: snap.StateVersion,
		TerminalReason: snap.TerminalReason,
		CreatedAt:      snap.CreatedAt, UpdatedAt: snap.UpdatedAt,
	}
	if snap.ProgressRef != nil {
		view.ProgressRef = string(*snap.ProgressRef)
	}
	if snap.ActivatedAt != nil {
		at := *snap.ActivatedAt
		view.ActivatedAt = &at
	}
	if snap.TerminalAt != nil {
		at := *snap.TerminalAt
		view.TerminalAt = &at
	}
	return view
}
