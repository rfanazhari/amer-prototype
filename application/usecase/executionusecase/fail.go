package executionusecase

import (
	"context"
	"errors"

	"github.com/rfanazhari/amer-prototype/application/dto"
	"github.com/rfanazhari/amer-prototype/application/repository/executionport"
	"github.com/rfanazhari/amer-prototype/application/service"
)

// Fail moves a CREATED or RUNNING execution to FAILED.
type Fail struct {
	Transactions executionport.Transactor
	Executions   executionport.ExecutionStore
	Commands     executionport.CommandStore
	Clock        service.Clock
}

var _ FailOperation = Fail{}

// errNotImplementedFail marks the operation as not yet implemented.
var errNotImplementedFail = errors.New("Fail is not implemented")

// Execute is not yet implemented.
func (u Fail) Execute(_ context.Context, _ FailInput) (*dto.ExecutionView, error) {
	return nil, errNotImplementedFail
}
