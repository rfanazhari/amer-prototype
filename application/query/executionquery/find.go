package executionquery

import (
	"context"
	"errors"

	"github.com/rfanazhari/amer-prototype/application/dto"
	"github.com/rfanazhari/amer-prototype/application/repository/executionport"
	"github.com/rfanazhari/amer-prototype/application/service"
	"github.com/rfanazhari/amer-prototype/pkg/errs"
)

// Find reads an execution by id without creating or changing state.
type Find struct{ Executions executionport.ExecutionStore }

var _ FindOperation = Find{}

// Execute returns the execution view, or EXL_NOT_FOUND when the id is absent.
func (q Find) Execute(ctx context.Context, executionID string) (*dto.ExecutionView, error) {
	snap, err := q.Executions.Find(ctx, executionID)
	if err != nil {
		return nil, mapReadError(err)
	}
	return dto.NewExecutionView(snap, ""), nil
}

// FindByAssignment reads the execution linked by the interview pin without creating one.
type FindByAssignment struct{ Executions executionport.ExecutionStore }

var _ FindByAssignmentOperation = FindByAssignment{}

// Execute returns the execution view for the assignment, or EXL_NOT_FOUND when absent.
func (q FindByAssignment) Execute(ctx context.Context, assignmentID string) (*dto.ExecutionView, error) {
	snap, err := q.Executions.FindByAssignment(ctx, assignmentID)
	if err != nil {
		return nil, mapReadError(err)
	}
	return dto.NewExecutionView(snap, assignmentID), nil
}

// mapReadError converts absence to EXL_NOT_FOUND and keeps every other failure technical.
func mapReadError(err error) error {
	if errors.Is(err, executionport.ErrNotFound) {
		return service.ErrExecutionNotFound
	}
	return errs.Technical{Cause: err}
}
