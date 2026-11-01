package executionusecase

import (
	"context"

	"github.com/rfanazhari/amer-prototype/application/dto"
	"github.com/rfanazhari/amer-prototype/application/repository/executionport"
	"github.com/rfanazhari/amer-prototype/application/service"
)

// Start admits or finds the one interview execution for an assignment.
type Start struct {
	Assignments executionport.AssignmentLock
	Executions  executionport.ExecutionStore
	Commands    executionport.CommandStore
	Clock       service.Clock
	// NewID mints the execution identity; nil uses a random UUID.
	NewID func() string
}

var _ StartOperation = Start{}

// Execute starts or returns the existing execution for the assignment.
func (u Start) Execute(ctx context.Context, input StartInput) (*dto.ExecutionView, error) {
	return nil, nil
}
