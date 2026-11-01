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

// Start admits or finds the one interview execution for an assignment.
type Start struct {
	Assignments executionport.AssignmentLock
	Executions  executionport.ExecutionStore
	Commands    executionport.CommandStore
	Clock       service.Clock
	// NewID mints the execution identity; nil uses a random UUID.
	NewID func() string
}

var _ StartOperation = Start{}

// Execute starts or returns the existing execution for the assignment.
// Shape errors return before any lock. A unique race on the pin is reconciled once by reading the winner.
func (u Start) Execute(ctx context.Context, input StartInput) (*dto.ExecutionView, error) {
	if !validUUID(input.AssignmentID) || !validToken(input.RequestKey) || !validToken(input.ActorRef) || !validRefs(input.InputRefs) {
		return nil, service.ErrExecutionInvalidInput
	}
	view, err := u.attempt(ctx, input)
	if errors.Is(err, executionport.ErrAlreadyExists) {
		view, err = u.attempt(ctx, input)
	}
	if err != nil {
		return nil, mapStoreError(err)
	}
	return view, nil
}

// attempt runs one assignment-locked transaction and returns the view only after it commits.
func (u Start) attempt(ctx context.Context, input StartInput) (*dto.ExecutionView, error) {
	var (
		view        *dto.ExecutionView
		unavailable bool
	)
	err := u.Assignments.WithAssignment(ctx, input.AssignmentID, func(ctx context.Context, row executionport.AssignmentRow) error {
		var err error
		view, unavailable, err = u.admitLocked(ctx, input, row)
		return err
	})
	if err != nil {
		return nil, err
	}
	if unavailable {
		return nil, service.ErrExecutionAssignmentUnavailable
	}
	return view, nil
}

// admitLocked decides the start outcome under the assignment lock. An unavailable assignment commits any
// required cancellation and reports unavailable so the caller returns the error after the commit.
func (u Start) admitLocked(ctx context.Context, input StartInput, row executionport.AssignmentRow) (*dto.ExecutionView, bool, error) {
	existing, err := u.Executions.FindByAssignment(ctx, input.AssignmentID)
	found := err == nil
	if err != nil && !errors.Is(err, executionport.ErrNotFound) {
		return nil, false, err
	}
	now := trustedNow(u.Clock)
	if row.AccessRevokedAt != nil {
		if found {
			if err := u.cancelNonterminal(ctx, existing, input.ActorRef, now); err != nil {
				return nil, false, err
			}
		}
		return nil, true, nil
	}
	if !found && row.CompletedAt != nil {
		return nil, true, nil
	}
	fingerprint := intentFingerprint(input.AssignmentID, row.InstanceID, row.RevisionID, input.InputRefs)
	binding := execution.Binding{InstanceID: row.InstanceID, RevisionID: row.RevisionID}
	if found {
		if !existing.Binding.Equal(binding) {
			return nil, false, executionport.ErrIdempotencyConflict
		}
		if _, err := u.Commands.FindOrInsert(ctx, input.AssignmentID, startCommandKind, input.RequestKey, fingerprint, existing.ID); err != nil {
			return nil, false, err
		}
		return dto.NewExecutionView(existing, input.AssignmentID), false, nil
	}
	_, snap, fact, err := execution.NewExecution(mintID(u.NewID), binding, now)
	if err != nil {
		return nil, false, err
	}
	fact.ActorRef = input.ActorRef
	if err := u.Executions.InsertInterview(ctx, input.AssignmentID, snap, fact); err != nil {
		return nil, false, err
	}
	if _, err := u.Commands.FindOrInsert(ctx, input.AssignmentID, startCommandKind, input.RequestKey, fingerprint, snap.ID); err != nil {
		return nil, false, err
	}
	return dto.NewExecutionView(snap, input.AssignmentID), false, nil
}

// cancelNonterminal cancels a CREATED or RUNNING execution in the open transaction and leaves a terminal one alone.
func (u Start) cancelNonterminal(ctx context.Context, existing execution.Snapshot, actorRef string, now time.Time) error {
	if existing.State != execution.StateCreated && existing.State != execution.StateRunning {
		return nil
	}
	return u.Executions.Update(ctx, existing.ID, func(snap *execution.Snapshot) (*execution.TransitionFact, error) {
		agg, err := execution.RestoreExecution(*snap)
		if err != nil {
			return nil, err
		}
		next, fact, err := agg.Cancel(accessRevokedReason, actorRef, now)
		if err != nil {
			return nil, err
		}
		*snap = next
		return &fact, nil
	})
}
