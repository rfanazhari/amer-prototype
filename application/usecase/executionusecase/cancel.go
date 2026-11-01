package executionusecase

import (
	"context"
	"errors"
	"time"

	"github.com/rfanazhari/amer-prototype/application/dto"
	"github.com/rfanazhari/amer-prototype/application/repository/executionport"
	"github.com/rfanazhari/amer-prototype/application/service"
	"github.com/rfanazhari/amer-prototype/domain/aggregate/execution"
)

const (
	// cancelCommandKind is the command kind that stores the identity of an early stop.
	cancelCommandKind = "cancel"
	// authorityExpiredReason is the terminal reason stored when the candidate's authority ended during a run.
	authorityExpiredReason = "authority_expired"
)

// Cancel moves a CREATED or RUNNING execution to CANCELLED for a confirmed early stop.
type Cancel struct {
	Transactions executionport.Transactor
	Executions   executionport.ExecutionStore
	Commands     executionport.CommandStore
	Clock        service.Clock
}

var _ CancelOperation = Cancel{}

// Execute cancels the execution once. The same cancellation repeated returns the stored outcome and any other
// terminal outcome is a conflict. A stale version writes nothing.
func (u Cancel) Execute(ctx context.Context, input CancelInput) (*dto.ExecutionView, error) {
	if !validUUID(input.ExecutionID) || !validToken(input.RequestKey) || !validToken(input.ActorRef) ||
		!validToken(input.Reason) || input.ExpectedVersion < 1 {
		return nil, service.ErrExecutionInvalidInput
	}
	now := trustedNow(u.Clock)
	return commitMutation(ctx, u.Transactions, u.Executions, input.ExecutionID,
		func(snap *execution.Snapshot) (*execution.TransitionFact, bool, error) {
			if terminalState(snap.State) {
				if snap.State == execution.StateCancelled && snap.TerminalReason == input.Reason {
					return nil, false, nil
				}
				return nil, false, service.ErrExecutionTerminalConflict
			}
			if snap.StateVersion != input.ExpectedVersion {
				return nil, false, service.ErrExecutionStaleVersion
			}
			agg, err := execution.RestoreExecution(*snap)
			if err != nil {
				return nil, false, err
			}
			next, fact, err := agg.Cancel(input.Reason, input.ActorRef, now)
			if err != nil {
				return nil, false, err
			}
			*snap = next
			return &fact, true, nil
		},
		func(ctx context.Context) error {
			fingerprint := commandFingerprint(input.ExecutionID, cancelCommandKind, "", input.Reason)
			_, err := u.Commands.FindOrInsert(ctx, input.ExecutionID, cancelCommandKind, input.RequestKey, fingerprint, input.ExecutionID)
			return err
		})
}

// CancelNonterminal cancels the nonterminal execution of an assignment whose access has ended.
type CancelNonterminal struct {
	Assignments executionport.AssignmentLock
	Executions  executionport.ExecutionStore
	Clock       service.Clock
}

var _ CancelNonterminalOperation = CancelNonterminal{}

// Execute locks the assignment, then the execution, and cancels it unless it is already terminal. The caller has
// already committed the access change and decided that authority expired; this operation reads neither the
// session clock nor the assignment's revocation marker. An assignment without an execution has nothing to cancel.
func (u CancelNonterminal) Execute(ctx context.Context, input CancelNonterminalInput) error {
	if !validUUID(input.AssignmentID) || !validToken(input.ActorRef) ||
		(input.Reason != accessRevokedReason && input.Reason != authorityExpiredReason) {
		return service.ErrExecutionInvalidInput
	}
	now := trustedNow(u.Clock)
	err := u.Assignments.WithAssignment(ctx, input.AssignmentID, func(ctx context.Context, _ executionport.AssignmentRow) error {
		existing, err := u.Executions.FindByAssignment(ctx, input.AssignmentID)
		if errors.Is(err, executionport.ErrNotFound) {
			return nil
		}
		if err != nil {
			return err
		}
		return cancelNonterminal(ctx, u.Executions, existing, input.Reason, input.ActorRef, now)
	})
	return mapStoreError(err)
}

// cancelNonterminal cancels a CREATED or RUNNING execution in the open transaction and leaves a terminal one alone.
// The locked row is checked again inside Update because a completion locks only the execution and can finish
// between the read under the assignment lock and the execution lock.
func cancelNonterminal(ctx context.Context, store executionport.ExecutionStore, existing execution.Snapshot,
	reason, actorRef string, now time.Time) error {
	if terminalState(existing.State) {
		return nil
	}
	return store.Update(ctx, existing.ID, func(snap *execution.Snapshot) (*execution.TransitionFact, error) {
		if terminalState(snap.State) {
			return nil, nil
		}
		agg, err := execution.RestoreExecution(*snap)
		if err != nil {
			return nil, err
		}
		next, fact, err := agg.Cancel(reason, actorRef, now)
		if err != nil {
			return nil, err
		}
		*snap = next
		return &fact, nil
	})
}
