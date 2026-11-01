package executionusecase

import (
	"context"

	"github.com/rfanazhari/amer-prototype/application/dto"
	"github.com/rfanazhari/amer-prototype/application/repository/executionport"
	"github.com/rfanazhari/amer-prototype/application/service"
	"github.com/rfanazhari/amer-prototype/domain/aggregate/execution"
)

// CommitProgress stores a committed progress reference while an execution is RUNNING.
type CommitProgress struct {
	Transactions executionport.Transactor
	Executions   executionport.ExecutionStore
	Clock        service.Clock
}

var _ CommitProgressOperation = CommitProgress{}

// Execute stores the reference and leaves the state RUNNING. It never completes the execution and, because the
// command kinds have no progress entry, records no command row.
func (u CommitProgress) Execute(ctx context.Context, input CommitProgressInput) (*dto.ExecutionView, error) {
	if !fencedShapeValid(input.ExecutionID, input.RequestKey, input.ActorRef, input.OperationKey, input.ExpectedVersion, input.Generation) ||
		!validToken(input.ProgressRef) {
		return nil, service.ErrExecutionInvalidInput
	}
	now := trustedNow(u.Clock)
	deadline := utcDeadline(input.Deadline)
	return commitMutation(ctx, u.Transactions, u.Executions, input.ExecutionID,
		func(snap *execution.Snapshot) (*execution.TransitionFact, bool, error) {
			agg, err := execution.RestoreExecution(*snap)
			if err != nil {
				return nil, false, err
			}
			next, err := agg.CommitProgress(execution.ProgressRef(input.ProgressRef), input.ExpectedVersion, input.Generation, deadline, now)
			if err != nil {
				return nil, false, err
			}
			*snap = next
			return nil, true, nil
		}, nil)
}
