package executionusecase

import (
	"context"

	"github.com/rfanazhari/amer-prototype/application/dto"
	"github.com/rfanazhari/amer-prototype/application/repository/executionport"
	"github.com/rfanazhari/amer-prototype/application/service"
	"github.com/rfanazhari/amer-prototype/domain/aggregate/execution"
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

// Execute completes the execution. The terminal row decides first: the same completion returns the stored
// outcome and any other terminal outcome is a conflict. Otherwise the domain checks version, fencing and state
// run before the progress port is read under the lock, so a stale call never reaches the port.
func (u Complete) Execute(ctx context.Context, input CompleteInput) (*dto.ExecutionView, error) {
	if !fencedShapeValid(input.ExecutionID, input.RequestKey, input.ActorRef, input.OperationKey, input.ExpectedVersion, input.Generation) ||
		!validToken(input.ProgressRef) {
		return nil, service.ErrExecutionInvalidInput
	}
	now := trustedNow(u.Clock)
	deadline := utcDeadline(input.Deadline)
	ref := execution.ProgressRef(input.ProgressRef)
	return commitMutation(ctx, u.Transactions, u.Executions, input.ExecutionID,
		func(snap *execution.Snapshot) (*execution.TransitionFact, bool, error) {
			if terminalState(snap.State) {
				if snap.State == execution.StateCompleted && snap.ProgressRef != nil && *snap.ProgressRef == ref {
					return nil, false, nil
				}
				return nil, false, service.ErrExecutionTerminalConflict
			}
			agg, err := execution.RestoreExecution(*snap)
			if err != nil {
				return nil, false, err
			}
			next, fact, err := agg.Complete(ref, input.ExpectedVersion, input.Generation, deadline, input.ActorRef, input.OperationKey, now)
			if err != nil {
				return nil, false, err
			}
			committed, err := u.Progress.IsCommitted(ctx, input.ExecutionID, ref)
			if err != nil {
				return nil, false, service.ErrExecutionDependencyUnavailable
			}
			if !committed {
				return nil, false, service.ErrExecutionProgressRequired
			}
			*snap = next
			return &fact, true, nil
		},
		func(ctx context.Context) error {
			fingerprint := commandFingerprint(input.ExecutionID, completeCommandKind, input.OperationKey, input.ProgressRef)
			_, err := u.Commands.FindOrInsert(ctx, input.ExecutionID, completeCommandKind, input.RequestKey, fingerprint, input.ExecutionID)
			return err
		})
}
