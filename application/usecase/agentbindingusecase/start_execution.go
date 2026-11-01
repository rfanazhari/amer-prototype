package agentbindingusecase

import (
	"context"
	"errors"

	"github.com/rfanazhari/amer-prototype/application/repository/agentbindingport"
	"github.com/rfanazhari/amer-prototype/application/repository/agentconfigport"
)

// StartExecution copies the assignment pin onto a started execution.
type StartExecution struct {
	Assignments agentbindingport.Assignments
	Executions  agentbindingport.Executions
	Revisions   agentconfigport.InstanceReader
}

// Execute is an inert RED stub; GREEN copies the assignment pin and ignores later publication.
func (u StartExecution) Execute(ctx context.Context, executionID, assignmentID string) (agentbindingport.ExecutionBinding, error) {
	_ = ctx
	_ = executionID
	_ = assignmentID
	_ = u.Assignments
	_ = u.Executions
	_ = u.Revisions
	return agentbindingport.ExecutionBinding{}, errors.New("start execution is not implemented")
}
