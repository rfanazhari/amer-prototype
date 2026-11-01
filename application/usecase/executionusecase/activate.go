package executionusecase

import (
	"context"

	"github.com/rfanazhari/amer-prototype/application/dto"
	"github.com/rfanazhari/amer-prototype/application/repository/executionport"
	"github.com/rfanazhari/amer-prototype/application/service"
	"github.com/rfanazhari/amer-prototype/domain/aggregate/execution"
)

// Activate moves a CREATED execution to RUNNING under the execution lock.
type Activate struct {
	Transactions executionport.Transactor
	Executions   executionport.ExecutionStore
	Commands     executionport.CommandStore
	Clock        service.Clock
}

var _ ActivateOperation = Activate{}

// Execute activates the same execution. A stale version, stale generation or exact deadline leaves it CREATED
// and writes nothing. A retry of the activation that already succeeded returns the RUNNING execution.
func (u Activate) Execute(ctx context.Context, input ActivateInput) (*dto.ExecutionView, error) {
	if !fencedShapeValid(input.ExecutionID, input.RequestKey, input.ActorRef, input.OperationKey, input.ExpectedVersion, input.Generation) {
		return nil, service.ErrExecutionInvalidInput
	}
	now := trustedNow(u.Clock)
	deadline := utcDeadline(input.Deadline)
	return commitMutation(ctx, u.Transactions, u.Executions, input.ExecutionID,
		func(snap *execution.Snapshot) (*execution.TransitionFact, bool, error) {
			if snap.State == execution.StateRunning && snap.CurrentGeneration == input.Generation && snap.StateVersion == input.ExpectedVersion+1 {
				return nil, false, nil
			}
			agg, err := execution.RestoreExecution(*snap)
			if err != nil {
				return nil, false, err
			}
			next, fact, err := agg.Activate(input.ExpectedVersion, input.ActorRef, input.OperationKey, input.Generation, deadline, now)
			if err != nil {
				return nil, false, err
			}
			*snap = next
			return &fact, true, nil
		},
		func(ctx context.Context) error {
			fingerprint := commandFingerprint(input.ExecutionID, activateCommandKind, input.OperationKey)
			_, err := u.Commands.FindOrInsert(ctx, input.ExecutionID, activateCommandKind, input.RequestKey, fingerprint, input.ExecutionID)
			return err
		})
}
