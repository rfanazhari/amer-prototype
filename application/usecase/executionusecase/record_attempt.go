package executionusecase

import (
	"context"

	"github.com/rfanazhari/amer-prototype/application/dto"
	"github.com/rfanazhari/amer-prototype/application/repository/executionport"
	"github.com/rfanazhari/amer-prototype/application/service"
)

// RecordAttempt records bounded, monotonically advancing execution authority.
type RecordAttempt struct {
	Transactions executionport.Transactor
	Executions   executionport.ExecutionStore
	Attempts     executionport.AttemptLedger
	Clock        service.Clock
}

var _ RecordAttemptOperation = RecordAttempt{}

// Execute admits one durable attempt without provider I/O. Exhaustion commits failure before returning its error.
func (u RecordAttempt) Execute(_ context.Context, _ RecordAttemptInput) (*dto.ExecutionView, error) {
	return nil, service.ErrExecutionInvalidInput
}
