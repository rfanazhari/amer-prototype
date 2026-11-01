package agentbindingusecase

import (
	"context"

	"github.com/rfanazhari/amer-prototype/application/repository/agentbindingport"
)

// PinAssignmentOperation pins a new assignment to the instance's current active revision.
type PinAssignmentOperation interface {
	// Execute stores the assignment pin selected from the current active revision.
	Execute(ctx context.Context, assignmentID, candidateID, instanceID string) (agentbindingport.AssignmentBinding, error)
}

// StartExecutionOperation copies an assignment pin onto a started execution.
type StartExecutionOperation interface {
	// Execute stores the execution pin from the assignment, ignoring later active revisions.
	Execute(ctx context.Context, executionID, assignmentID string) (agentbindingport.ExecutionBinding, error)
}

// RequirementsOperation restores immutable published requirements for a started execution.
type RequirementsOperation interface {
	// Execute returns the pinned revision identity and document required fields.
	Execute(ctx context.Context, executionID string) (Requirements, error)
}

// Requirements are the immutable published fields an execution must continue to use.
type Requirements struct {
	RevisionID     string
	InstanceID     string
	RequiredFields []string
}
