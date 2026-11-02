// Package accesslifecycle adapts the execution owner's access-ended cancellation for Identity and Access.
package accesslifecycle

import (
	"context"

	"github.com/rfanazhari/amer-prototype/application/repository/executionport"
	"github.com/rfanazhari/amer-prototype/application/service"
	"github.com/rfanazhari/amer-prototype/application/usecase/executionusecase"
)

// Canceller asks Execution Lifecycle to cancel a nonterminal execution after assignment access ended.
type Canceller struct {
	Assignments executionport.AssignmentLock
	Executions  executionport.ExecutionStore
	Clock       service.Clock
}

var _ executionusecase.CancelNonterminalOperation = Canceller{}

// Execute delegates to the lifecycle's own assignment-then-execution lock; it adds no policy of its own.
func (c Canceller) Execute(ctx context.Context, input executionusecase.CancelNonterminalInput) error {
	return executionusecase.CancelNonterminal{Assignments: c.Assignments, Executions: c.Executions, Clock: c.Clock}.Execute(ctx, input)
}
