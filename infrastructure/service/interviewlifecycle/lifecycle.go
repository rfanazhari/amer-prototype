// Package interviewlifecycle maps interview attempt UUIDs to stage-independent execution authority.
package interviewlifecycle

import (
	"context"
	"errors"
	"strconv"
	"time"

	"github.com/rfanazhari/amer-prototype/application/repository/executionport"
	"github.com/rfanazhari/amer-prototype/application/service"
	"github.com/rfanazhari/amer-prototype/application/usecase/executionusecase"
	"github.com/rfanazhari/amer-prototype/domain/aggregate/execution"
	"github.com/rfanazhari/amer-prototype/domain/interview"
)

// Lifecycle supplies the real execution fence and durable admission to the interview workflow.
type Lifecycle struct {
	Transactions executionport.Transactor
	Executions   executionport.ExecutionStore
	Attempts     executionport.AttemptLedger
	Commands     executionport.CommandStore
	Progress     executionport.ProgressCommitment
	Clock        service.Clock
}

var _ service.InterviewLifecycle = Lifecycle{}

// mapped translates lifecycle refusal to the interview's existing conflict vocabulary.
func mapped(err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, executionport.ErrAttemptBudget):
		return errors.Join(interview.ErrBudget, err)
	case errors.Is(err, execution.ErrStaleAttempt), errors.Is(err, execution.ErrStaleVersion), errors.Is(err, execution.ErrTerminalConflict), errors.Is(err, executionport.ErrNotFound), errors.Is(err, service.ErrExecutionStaleAttempt), errors.Is(err, service.ErrExecutionStaleVersion), errors.Is(err, service.ErrExecutionTerminalConflict), errors.Is(err, service.ErrExecutionInvalidTransition):
		return interview.ErrConflict
	default:
		return err
	}
}

// Activate reserves stage-independent attempt authority under the frozen initial or recovery limits.
func (a Lifecycle) Activate(ctx context.Context, r service.InterviewActivation) (service.InterviewActivationResult, error) {
	b := service.DefaultInterviewBudgets()
	if r.Budgets != b {
		return service.InterviewActivationResult{}, interview.ErrInvalid
	}
	snap, err := a.Executions.Find(ctx, r.Authority.ExecutionID)
	if err != nil {
		return service.InterviewActivationResult{}, mapped(err)
	}
	in := executionusecase.RecordAttemptInput{ExecutionID: r.Authority.ExecutionID, AttemptID: r.Authority.AttemptID, ExpectedVersion: r.Authority.ExpectedVersion, RequestKey: "attempt", ActorRef: "interview-runtime", OperationKey: "initial", Limit: b.ActivationAttempts, Window: b.ActivationWindow, AttemptTimeout: b.ActivationAttemptTimeout}
	if snap.ActivatedAt != nil {
		end := snap.ActivatedAt.Add(15 * time.Minute)
		in.HardDeadline = &end
	}
	switch r.Intent {
	case "activate":
	case "reconnect":
		if r.DetectedAt == nil {
			return service.InterviewActivationResult{}, interview.ErrConflict
		}
		in.OperationKey = "recovery:" + strconv.FormatInt(r.DetectedAt.UnixMilli(), 10)
		in.WindowStart = *r.DetectedAt
		in.Window = b.ReconnectWindow
		in.Limit = b.ReconnectAttempts
		in.AttemptTimeout = b.ReconnectWindow
	default:
		return service.InterviewActivationResult{}, interview.ErrInvalid
	}
	view, err := (executionusecase.RecordAttempt{Transactions: a.Transactions, Executions: a.Executions, Attempts: a.Attempts, Clock: a.Clock}).Execute(ctx, in)
	if err != nil {
		return service.InterviewActivationResult{}, mapped(err)
	}
	if view.ActivatedAt == nil {
		return service.InterviewActivationResult{}, interview.ErrRestore
	}
	return service.InterviewActivationResult{Authority: service.InterviewAttempt{ExecutionID: view.ID, AttemptID: r.Authority.AttemptID, ExpectedVersion: view.StateVersion}, RunningAt: *view.ActivatedAt, HardDeadline: view.ActivatedAt.Add(15 * time.Minute)}, nil
}

// within checks a current UUID, version and generation under the execution lock before any sibling write.
func (a Lifecycle) within(ctx context.Context, r service.InterviewAttempt, maximum time.Duration, write func(context.Context) error) error {
	guard := executionusecase.FencedWrite{Transactions: a.Transactions, Executions: a.Executions, Attempts: a.Attempts, Clock: a.Clock}
	return mapped(guard.Within(ctx, r.ExecutionID, r.AttemptID, r.ExpectedVersion, maximum, write))
}

// Fence checks current authority without authorizing a subsequent independent write.
func (a Lifecycle) Fence(ctx context.Context, r service.InterviewAttempt) error {
	return a.within(ctx, r, 0, func(context.Context) error { return nil })
}

// WithProgress locks execution before progress and joins all writes to one transaction.
func (a Lifecycle) WithProgress(ctx context.Context, r service.InterviewAttempt, write func(context.Context) error) error {
	return a.within(ctx, r, 15*time.Minute, write)
}

// Complete checks committed progress and current authority before the first terminal commit.
func (a Lifecycle) Complete(ctx context.Context, r service.InterviewCompletion) error {
	return a.within(ctx, r.Authority, 0, func(tx context.Context) error {
		attempt, err := a.Attempts.Find(tx, r.Authority.ExecutionID, r.Authority.AttemptID)
		if err != nil {
			return err
		}
		_, err = (executionusecase.Complete{Transactions: a.Transactions, Executions: a.Executions, Commands: a.Commands, Progress: a.Progress, Clock: a.Clock}).Execute(tx, executionusecase.CompleteInput{ExecutionID: r.Authority.ExecutionID, ExpectedVersion: r.Authority.ExpectedVersion, Generation: attempt.Generation, RequestKey: r.Authority.AttemptID, OperationKey: attempt.OperationKey, ActorRef: "interview-runtime", ProgressRef: r.ProgressReference})
		return err
	})
}

// Fail preserves progress and remains available when the hard deadline has been reached.
func (a Lifecycle) Fail(ctx context.Context, r service.InterviewFailure) error {
	return a.within(ctx, r.Authority, 0, func(tx context.Context) error {
		attempt, err := a.Attempts.Find(tx, r.Authority.ExecutionID, r.Authority.AttemptID)
		if err != nil {
			return err
		}
		_, err = (executionusecase.Fail{Transactions: a.Transactions, Executions: a.Executions, Commands: a.Commands, Clock: a.Clock}).Execute(tx, executionusecase.FailInput{ExecutionID: r.Authority.ExecutionID, ExpectedVersion: r.Authority.ExpectedVersion, Generation: attempt.Generation, RequestKey: r.Authority.AttemptID, OperationKey: attempt.OperationKey, ActorRef: "interview-runtime", Reason: r.Reason})
		return err
	})
}
