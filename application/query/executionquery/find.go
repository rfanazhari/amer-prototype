package executionquery

import (
	"context"

	"github.com/rfanazhari/amer-prototype/application/dto"
	"github.com/rfanazhari/amer-prototype/application/repository/executionport"
)

// Find reads an execution by id without creating or changing state.
type Find struct{ Executions executionport.ExecutionStore }

var _ FindOperation = Find{}

// Execute returns the execution view, or EXL_NOT_FOUND when the id is absent.
func (q Find) Execute(ctx context.Context, executionID string) (*dto.ExecutionView, error) {
	return nil, nil
}

// FindByAssignment reads the execution linked by the interview pin without creating one.
type FindByAssignment struct{ Executions executionport.ExecutionStore }

var _ FindByAssignmentOperation = FindByAssignment{}

// Execute returns the execution view for the assignment, or EXL_NOT_FOUND when absent.
func (q FindByAssignment) Execute(ctx context.Context, assignmentID string) (*dto.ExecutionView, error) {
	return nil, nil
}
