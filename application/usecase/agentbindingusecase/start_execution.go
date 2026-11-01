package agentbindingusecase

import (
	"context"

	"github.com/rfanazhari/amer-prototype/application/repository/agentbindingport"
	"github.com/rfanazhari/amer-prototype/application/repository/agentconfigport"
)

// StartExecution copies the assignment pin onto a started execution.
type StartExecution struct {
	Assignments agentbindingport.Assignments
	Executions  agentbindingport.Executions
	Revisions   agentconfigport.InstanceReader
}

// Execute stores the execution pin from the assignment, ignoring later active revisions.
func (u StartExecution) Execute(ctx context.Context, executionID, assignmentID string) (agentbindingport.ExecutionBinding, error) {
	assignment, err := u.Assignments.Find(ctx, assignmentID)
	if err != nil {
		return agentbindingport.ExecutionBinding{}, err
	}
	if _, err := u.Revisions.FindRevision(ctx, assignment.InstanceID, assignment.RevisionID); err != nil {
		return agentbindingport.ExecutionBinding{}, err
	}
	binding := agentbindingport.ExecutionBinding{
		ExecutionID: executionID, AssignmentID: assignment.AssignmentID, InstanceID: assignment.InstanceID, RevisionID: assignment.RevisionID,
	}
	if err := u.Executions.Insert(ctx, binding); err != nil {
		return agentbindingport.ExecutionBinding{}, err
	}
	return binding, nil
}
