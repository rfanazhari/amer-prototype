package executionusecase

import (
	"context"
	"errors"

	"github.com/rfanazhari/amer-prototype/application/dto"
	"github.com/rfanazhari/amer-prototype/application/repository/executionport"
	"github.com/rfanazhari/amer-prototype/application/service"
)

// Complete moves a RUNNING execution to COMPLETED once the progress port confirms committed progress.
type Complete struct {
	Transactions executionport.Transactor
	Executions   executionport.ExecutionStore
	Commands     executionport.CommandStore
	Progress     executionport.ProgressCommitment
	Clock        service.Clock
}

var _ CompleteOperation = Complete{}

// errNotImplementedComplete marks the operation as not yet implemented.
var errNotImplementedComplete = errors.New("Complete is not implemented")

// Execute is not yet implemented.
func (u Complete) Execute(_ context.Context, _ CompleteInput) (*dto.ExecutionView, error) {
	return nil, errNotImplementedComplete
}
