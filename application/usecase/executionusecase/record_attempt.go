package executionusecase

import (
	"context"

	"github.com/rfanazhari/amer-prototype/application/dto"
	"github.com/rfanazhari/amer-prototype/application/repository/executionport"
	"github.com/rfanazhari/amer-prototype/application/service"
	"github.com/rfanazhari/amer-prototype/domain/aggregate/execution"
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
func (u RecordAttempt) Execute(ctx context.Context, in RecordAttemptInput) (*dto.ExecutionView, error) {
	if !validUUID(in.ExecutionID) || !validUUID(in.AttemptID) || !validToken(in.OperationKey) || !validToken(in.ActorRef) || in.ExpectedVersion < 1 || in.Limit < 1 || in.Window <= 0 || in.AttemptTimeout <= 0 {
		return nil, service.ErrExecutionInvalidInput
	}
	now := trustedNow(u.Clock)
	var final execution.Snapshot
	exhausted := false
	err := u.Transactions.WithTransaction(ctx, func(tx context.Context) error {
		return u.Executions.Update(tx, in.ExecutionID, func(snap *execution.Snapshot) (*execution.TransitionFact, error) {
			if terminalState(snap.State) {
				return nil, execution.ErrTerminalConflict
			}
			if snap.StateVersion != in.ExpectedVersion {
				return nil, execution.ErrStaleVersion
			}
			history, err := u.Attempts.List(tx, in.ExecutionID, in.OperationKey)
			if err != nil {
				return nil, err
			}
			start := in.WindowStart
			if start.IsZero() {
				start = now
				if len(history) > 0 {
					start = history[0].CreatedAt
				}
			}
			end := start.Add(in.Window)
			if in.HardDeadline != nil && in.HardDeadline.Before(end) {
				end = *in.HardDeadline
			}
			agg, err := execution.RestoreExecution(*snap)
			if err != nil {
				return nil, err
			}
			if len(history) > 0 {
				last := history[len(history)-1]
				if last.Outcome == "IN_PROGRESS" && now.Before(last.Deadline) {
					return nil, execution.ErrStaleAttempt
				}
			}
			if len(history) >= in.Limit || !now.Before(end) || now.Before(start) {
				next, fact, failErr := agg.Fail("attempt_budget_exhausted", snap.StateVersion, snap.CurrentGeneration, nil, in.ActorRef, in.OperationKey, now)
				if failErr != nil {
					return nil, failErr
				}
				*snap = next
				final = next
				exhausted = true
				return &fact, nil
			}

			gen := snap.CurrentGeneration + 1
			if in.Generation != 0 && in.Generation != gen {
				return nil, execution.ErrStaleAttempt
			}
			var fact *execution.TransitionFact
			if snap.State == execution.StateCreated {
				next, f, activateErr := agg.Activate(in.ExpectedVersion, in.ActorRef, in.OperationKey, gen, nil, now)
				if activateErr != nil {
					return nil, activateErr
				}
				*snap = next
				fact = &f
			} else {
				next, replaceErr := agg.RecordAttempt(in.ExpectedVersion, gen, now)
				if replaceErr != nil {
					return nil, replaceErr
				}
				*snap = next
			}
			deadline := now.Add(in.AttemptTimeout)
			if end.Before(deadline) {
				deadline = end
			}
			if err = u.Attempts.Insert(tx, executionport.Attempt{ID: in.AttemptID, ExecutionID: in.ExecutionID, OperationKey: in.OperationKey, Generation: gen, Number: len(history) + 1, Outcome: "IN_PROGRESS", CreatedAt: now, Deadline: deadline}); err != nil {
				return nil, err
			}
			final = *snap
			return fact, nil
		})
	})
	if err != nil {
		return nil, mapLifecycleError(err)
	}
	if exhausted {
		return nil, executionport.ErrAttemptBudget
	}
	return dto.NewExecutionView(final, ""), nil
}
