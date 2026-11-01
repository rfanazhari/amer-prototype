package dto

import "github.com/rfanazhari/amer-prototype/domain/aggregate/execution"

// NewExecutionView copies a committed snapshot into a detached view.
// assignmentID is empty for an execution that has no interview pin.
func NewExecutionView(snap execution.Snapshot, assignmentID string) *ExecutionView {
	return nil
}
