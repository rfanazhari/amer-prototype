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
	// activateCommandKind is the command kind that stores the identity of an activation.
	activateCommandKind = "activate"
	// completeCommandKind is the command kind that stores the identity of a completion.
	completeCommandKind = "complete"
	// failCommandKind is the command kind that stores the identity of a failure.
	failCommandKind = "fail"
)

// fencedShapeValid reports whether the fields shared by every fenced lifecycle command are well formed.
func fencedShapeValid(executionID, requestKey, actorRef, operationKey string, expectedVersion, generation int64) bool {
	return validUUID(executionID) && validToken(requestKey) && validToken(actorRef) && validToken(operationKey) &&
		expectedVersion >= 1 && generation >= 1
}

// utcDeadline returns a detached UTC millisecond copy of an optional deadline.
func utcDeadline(deadline *time.Time) *time.Time {
	if deadline == nil {
		return nil
	}
	value := deadline.UTC().Truncate(time.Millisecond)
	return &value
}

// terminalState reports whether the state is a terminal outcome.
func terminalState(state execution.State) bool {
	return state == execution.StateCompleted || state == execution.StateFailed || state == execution.StateCancelled
}

// commandFingerprint hashes the execution, the command kind and the fields that define one command's intent.
func commandFingerprint(executionID, kind, operationKey string, fields ...string) string {
	return intentFingerprint(executionID, kind, operationKey, fields)
}

// mapLifecycleError converts a domain sentinel or a port failure to its public execution error.
func mapLifecycleError(err error) error {
	switch {
	case errors.Is(err, execution.ErrInvalidTransition):
		return service.ErrExecutionInvalidTransition
	case errors.Is(err, execution.ErrStaleVersion):
		return service.ErrExecutionStaleVersion
	case errors.Is(err, execution.ErrStaleAttempt):
		return service.ErrExecutionStaleAttempt
	case errors.Is(err, execution.ErrProgressRequired):
		return service.ErrExecutionProgressRequired
	case errors.Is(err, execution.ErrTerminalConflict):
		return service.ErrExecutionTerminalConflict
	}
	return mapStoreError(err)
}

// applyFunc mutates the locked snapshot and reports the transition fact to append and whether anything changed.
type applyFunc func(snap *execution.Snapshot) (*execution.TransitionFact, bool, error)

// commitMutation runs apply under the execution lock in one transaction and records the command row only when
// the row changed. A replay that changed nothing writes nothing. Errors are mapped after the transaction ends.
func commitMutation(ctx context.Context, tx executionport.Transactor, store executionport.ExecutionStore, id string,
	apply applyFunc, record func(ctx context.Context) error) (*dto.ExecutionView, error) {
	var final execution.Snapshot
	err := tx.WithTransaction(ctx, func(ctx context.Context) error {
		changed := false
		if err := store.Update(ctx, id, func(snap *execution.Snapshot) (*execution.TransitionFact, error) {
			fact, did, err := apply(snap)
			if err != nil {
				return nil, err
			}
			changed, final = did, *snap
			return fact, nil
		}); err != nil {
			return err
		}
		if changed && record != nil {
			return record(ctx)
		}
		return nil
	})
	if err != nil {
		return nil, mapLifecycleError(err)
	}
	return dto.NewExecutionView(final, ""), nil
}
