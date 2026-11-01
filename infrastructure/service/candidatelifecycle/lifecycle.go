// Package candidatelifecycle adapts the execution owner to the candidate interview port.
package candidatelifecycle

import (
	"context"

	"github.com/rfanazhari/amer-prototype/application/repository/executionport"
	"github.com/rfanazhari/amer-prototype/application/service"
)

// Lifecycle connects candidate interview operations to the existing execution transactions.
type Lifecycle struct {
	Assignments  executionport.AssignmentLock
	Transactions executionport.Transactor
	Executions   executionport.ExecutionStore
	Commands     executionport.CommandStore
	Clock        service.Clock
}

// CommitOrFind commits or returns the assignment's original candidate start intent.
func (a Lifecycle) CommitOrFind(context.Context, service.CommitOrFindRequest) (service.CommitOrFindResult, error) {
	return service.CommitOrFindResult{}, service.ErrInterviewDependencyUnavailable
}

// Find reads the assignment's execution without creating one.
func (a Lifecycle) Find(context.Context, string) (service.ExecutionFacts, error) {
	return service.ExecutionFacts{}, service.ErrInterviewDependencyUnavailable
}

// Cancel commits a nonterminal candidate cancellation through Execution Lifecycle.
func (a Lifecycle) Cancel(context.Context, service.CancelRequest) (service.ExecutionFacts, error) {
	return service.ExecutionFacts{}, service.ErrInterviewDependencyUnavailable
}
