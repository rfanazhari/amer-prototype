package executionusecase

import (
	"context"
	"errors"

	"github.com/rfanazhari/amer-prototype/application/dto"
	"github.com/rfanazhari/amer-prototype/application/repository/executionport"
	"github.com/rfanazhari/amer-prototype/application/service"
)

// Activate moves a CREATED execution to RUNNING under the execution lock.
type Activate struct {
	Transactions executionport.Transactor
	Executions   executionport.ExecutionStore
	Commands     executionport.CommandStore
	Clock        service.Clock
}

var _ ActivateOperation = Activate{}

// errNotImplementedActivate marks the operation as not yet implemented.
var errNotImplementedActivate = errors.New("Activate is not implemented")

// Execute is not yet implemented.
func (u Activate) Execute(_ context.Context, _ ActivateInput) (*dto.ExecutionView, error) {
	return nil, errNotImplementedActivate
}
