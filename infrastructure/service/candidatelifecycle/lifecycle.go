// Package candidatelifecycle adapts the execution owner to the candidate interview port.
package candidatelifecycle

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/rfanazhari/amer-prototype/application/dto"
	"github.com/rfanazhari/amer-prototype/application/query/executionquery"
	"github.com/rfanazhari/amer-prototype/application/repository/executionport"
	"github.com/rfanazhari/amer-prototype/application/service"
	"github.com/rfanazhari/amer-prototype/application/usecase/executionusecase"
	"github.com/rfanazhari/amer-prototype/domain/aggregate/candidateinterview"
	"github.com/rfanazhari/amer-prototype/domain/aggregate/execution"
)

// Lifecycle connects candidate interview operations to the existing execution transactions.
type Lifecycle struct {
	Assignments  executionport.AssignmentLock
	Transactions executionport.Transactor
	Executions   executionport.ExecutionStore
	Commands     executionport.CommandStore
	Clock        service.Clock
}

var _ service.ExecutionLifecycle = Lifecycle{}

// interviewError translates sibling policy failures without exposing its public error namespace.
func interviewError(err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, service.ErrExecutionNotFound):
		return service.ErrInterviewNotFound
	case errors.Is(err, service.ErrExecutionIdempotencyConflict), errors.Is(err, service.ErrExecutionStaleVersion), errors.Is(err, service.ErrExecutionTerminalConflict), errors.Is(err, service.ErrExecutionAssignmentUnavailable):
		return service.ErrInterviewConflict
	case errors.Is(err, service.ErrExecutionUncertainCommit):
		return service.ErrInterviewStartUncertain
	case errors.Is(err, service.ErrExecutionDependencyUnavailable), errors.Is(err, executionport.ErrDependencyUnavailable):
		return service.ErrInterviewDependencyUnavailable
	default:
		return err
	}
}

// facts copies only candidate-safe lifecycle fields and derives the stable deadline from persisted activation.
func facts(view *dto.ExecutionView) service.ExecutionFacts {
	result := service.ExecutionFacts{ID: view.ID, AssignmentID: view.AssignmentID, State: view.State, Version: view.StateVersion}
	if view.State == string(execution.StateRunning) && view.ActivatedAt != nil {
		deadline := view.ActivatedAt.UTC().Truncate(time.Millisecond).Add(candidateinterview.InterviewBudget)
		result.HardDeadline = &deadline
	}
	return result
}

// CommitOrFind uses one command identity per assignment and hashes typed intent references to fit the
// lifecycle's 64-character token limit while retaining the candidate API's 128-character key contract.
// The id minted inside this call identifies a committed insert; a replay or reconciled race returns another id.
func (a Lifecycle) CommitOrFind(ctx context.Context, req service.CommitOrFindRequest) (service.CommitOrFindResult, error) {
	var minted string
	start := executionusecase.Start{Assignments: a.Assignments, Executions: a.Executions, Commands: a.Commands, Clock: a.Clock, NewID: func() string { minted = uuid.NewString(); return minted }}
	view, err := start.Execute(ctx, executionusecase.StartInput{
		AssignmentID: req.AssignmentID, RequestKey: "candidate-start", ActorRef: "candidate-interview",
		InputRefs: []string{fmt.Sprintf("%x", sha256.Sum256([]byte("key:"+req.IdempotencyKey))), fmt.Sprintf("%x", sha256.Sum256([]byte("consent:"+req.ConsentID)))},
	})
	if err != nil {
		return service.CommitOrFindResult{}, interviewError(err)
	}
	return service.CommitOrFindResult{Execution: facts(view), Created: minted != "" && view.ID == minted}, nil
}

// Find reads through the assignment pin without creating or changing any execution.
func (a Lifecycle) Find(ctx context.Context, assignmentID string) (service.ExecutionFacts, error) {
	view, err := (executionquery.FindByAssignment{Executions: a.Executions}).Execute(ctx, assignmentID)
	if err != nil {
		return service.ExecutionFacts{}, interviewError(err)
	}
	return facts(view), nil
}

// nonterminalStore narrows lifecycle's replayable cancel to the candidate contract under the same row lock.
type nonterminalStore struct{ executionport.ExecutionStore }

// Update refuses a terminal outcome before lifecycle's mutation callback, closing the read/cancel race.
func (s nonterminalStore) Update(ctx context.Context, id string, fn func(*execution.Snapshot) (*execution.TransitionFact, error)) error {
	return s.ExecutionStore.Update(ctx, id, func(snap *execution.Snapshot) (*execution.TransitionFact, error) {
		switch snap.State {
		case execution.StateCompleted, execution.StateFailed, execution.StateCancelled:
			return nil, service.ErrExecutionTerminalConflict
		}
		return fn(snap)
	})
}

// Cancel commits through lifecycle; its returned view is exposed only after the transaction commits.
func (a Lifecycle) Cancel(ctx context.Context, req service.CancelRequest) (service.ExecutionFacts, error) {
	cancel := executionusecase.Cancel{Transactions: a.Transactions, Executions: nonterminalStore{ExecutionStore: a.Executions}, Commands: a.Commands, Clock: a.Clock}
	view, err := cancel.Execute(ctx, executionusecase.CancelInput{ExecutionID: req.ExecutionID, ExpectedVersion: req.ExpectedVersion, Reason: req.Reason, ActorRef: "candidate-interview", RequestKey: fmt.Sprintf("candidate-cancel:%d", req.ExpectedVersion)})
	if err != nil {
		return service.ExecutionFacts{}, interviewError(err)
	}
	return facts(view), nil
}
