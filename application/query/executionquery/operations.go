package executionquery

import (
	"context"

	"github.com/rfanazhari/amer-prototype/application/dto"
)

// FindOperation returns a stored execution by its stable identity.
// It does not create or mutate state. Absence is an error.
type FindOperation interface {
	// Execute returns the execution view, or an error when the id is absent.
	Execute(ctx context.Context, executionID string) (*dto.ExecutionView, error)
}

// FindByAssignmentOperation returns the execution linked by the interview assignment pin.
// It does not create or mutate state. Absence is an error.
type FindByAssignmentOperation interface {
	// Execute returns the execution view for the assignment, or an error when absent.
	Execute(ctx context.Context, assignmentID string) (*dto.ExecutionView, error)
}
