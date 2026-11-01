package executionusecase

import (
	"context"

	"github.com/rfanazhari/amer-prototype/application/dto"
	"github.com/rfanazhari/amer-prototype/application/repository/executionport"
	"github.com/rfanazhari/amer-prototype/application/service"
	"github.com/rfanazhari/amer-prototype/domain/aggregate/execution"
)

// Fail moves a CREATED or RUNNING execution to FAILED.
type Fail struct {
	Transactions executionport.Transactor
	Executions   executionport.ExecutionStore
	Commands     executionport.CommandStore
	Clock        service.Clock
}

var _ FailOperation = Fail{}

// Execute fails the execution once. The same failure repeated returns the stored outcome and any other
// terminal outcome is a conflict. Partial progress already stored stays on the row.
func (u Fail) Execute(ctx context.Context, input FailInput) (*dto.ExecutionView, error) {
	if !fencedShapeValid(input.ExecutionID, input.RequestKey, input.ActorRef, input.OperationKey, input.ExpectedVersion, input.Generation) ||
		!validToken(input.Reason) {
		return nil, service.ErrExecutionInvalidInput
	}
	now := trustedNow(u.Clock)
	deadline := utcDeadline(input.Deadline)
	return commitMutation(ctx, u.Transactions, u.Executions, input.ExecutionID,
		func(snap *execution.Snapshot) (*execution.TransitionFact, bool, error) {
			if terminalState(snap.State) {
				if snap.State == execution.StateFailed && snap.TerminalReason == input.Reason {
					return nil, false, nil
				}
				return nil, false, service.ErrExecutionTerminalConflict
			}
			agg, err := execution.RestoreExecution(*snap)
			if err != nil {
				return nil, false, err
			}
			next, fact, err := agg.Fail(input.Reason, input.ExpectedVersion, input.Generation, deadline, input.ActorRef, input.OperationKey, now)
			if err != nil {
				return nil, false, err
			}
			*snap = next
			return &fact, true, nil
		},
		func(ctx context.Context) error {
			fingerprint := commandFingerprint(input.ExecutionID, failCommandKind, input.OperationKey, input.Reason)
			_, err := u.Commands.FindOrInsert(ctx, input.ExecutionID, failCommandKind, input.RequestKey, fingerprint, input.ExecutionID)
			return err
		})
}
