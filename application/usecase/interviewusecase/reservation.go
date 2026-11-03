package interviewusecase

import (
	"context"
	"errors"
	"time"

	"github.com/rfanazhari/amer-prototype/application/repository/executionport"
)

// prepare keeps durable admission and the scoped shell reservation in one transaction before I/O.
// Port-only workflow callers can supply their own transaction through the lifecycle mock.
func (u Workflow) prepare(ctx context.Context, write func(context.Context) error) error {
	if u.Transactions == nil {
		return write(ctx)
	}
	var budgetErr error
	err := u.Transactions.WithTransaction(ctx, func(tx context.Context) error {
		err := write(tx)
		if errors.Is(err, executionport.ErrAttemptBudget) {
			budgetErr = err
			return nil
		}
		return err
	})
	if err != nil {
		return err
	}
	return budgetErr
}

// reserve commits the HTTP command identity with its shell, after lifecycle version advancement.
func (u Workflow) reserve(ctx context.Context, req SessionRequest) error {
	if req.Command == nil {
		return nil
	}
	command := *req.Command
	command.ExecutionVersion = req.Authority.ExpectedVersion
	return u.Commands.Insert(ctx, command)
}

// failed records a known setup failure while retaining the non-retryable key and shell.
func (u Workflow) failed(ctx context.Context, req SessionRequest) {
	if req.Command == nil {
		return
	}
	cleanup, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
	defer cancel()
	_ = u.Commands.Fail(cleanup, req.Authority.ExecutionID, req.IdempotencyKey)
}

// finalize joins original replay fields to the successful provider/progress result.
func (u Workflow) finalize(ctx context.Context, req SessionRequest, sdp string) error {
	if req.Command == nil {
		return nil
	}
	command := *req.Command
	command.State = "succeeded"
	command.AnswerSDP = sdp
	command.ExecutionVersion = req.Authority.ExpectedVersion
	return u.Commands.Finalize(ctx, command)
}
