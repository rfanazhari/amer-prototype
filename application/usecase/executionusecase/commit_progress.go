package executionusecase

import (
	"context"
	"errors"

	"github.com/rfanazhari/amer-prototype/application/dto"
	"github.com/rfanazhari/amer-prototype/application/repository/executionport"
	"github.com/rfanazhari/amer-prototype/application/service"
)

// CommitProgress stores a committed progress reference while an execution is RUNNING.
type CommitProgress struct {
	Transactions executionport.Transactor
	Executions   executionport.ExecutionStore
	Clock        service.Clock
}

var _ CommitProgressOperation = CommitProgress{}

// errNotImplementedCommitProgress marks the operation as not yet implemented.
var errNotImplementedCommitProgress = errors.New("CommitProgress is not implemented")

// Execute is not yet implemented.
func (u CommitProgress) Execute(_ context.Context, _ CommitProgressInput) (*dto.ExecutionView, error) {
	return nil, errNotImplementedCommitProgress
}
