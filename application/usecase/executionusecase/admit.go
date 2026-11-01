package executionusecase

import (
	"context"

	"github.com/rfanazhari/amer-prototype/application/dto"
	"github.com/rfanazhari/amer-prototype/application/repository/executionport"
	"github.com/rfanazhari/amer-prototype/application/service"
)

// Admit commits a CREATED execution for a caller-resolved binding without an assignment or pin.
type Admit struct {
	Transactions executionport.Transactor
	Executions   executionport.ExecutionStore
	Commands     executionport.CommandStore
	Clock        service.Clock
	// NewID mints the execution identity; nil uses a random UUID.
	NewID func() string
}

var _ AdmitOperation = Admit{}

// Execute admits a new execution or returns the existing one for the scope and key.
func (u Admit) Execute(ctx context.Context, input AdmitInput) (*dto.ExecutionView, error) {
	return nil, nil
}
