package executionusecase

import (
	"context"
	"errors"

	"github.com/rfanazhari/amer-prototype/application/dto"
	"github.com/rfanazhari/amer-prototype/application/repository/executionport"
	"github.com/rfanazhari/amer-prototype/application/service"
)

// errCancelNotImplemented marks a cancellation operation whose behavior has not been written yet.
var errCancelNotImplemented = errors.New("cancellation is not implemented")

// Cancel moves a CREATED or RUNNING execution to CANCELLED for a confirmed early stop.
type Cancel struct {
	Transactions executionport.Transactor
	Executions   executionport.ExecutionStore
	Commands     executionport.CommandStore
	Clock        service.Clock
}

var _ CancelOperation = Cancel{}

// Execute is an inert stub until the cancellation behavior is implemented.
func (u Cancel) Execute(_ context.Context, _ CancelInput) (*dto.ExecutionView, error) {
	return nil, errCancelNotImplemented
}

// CancelNonterminal cancels the nonterminal execution of an assignment whose access has ended.
type CancelNonterminal struct {
	Assignments executionport.AssignmentLock
	Executions  executionport.ExecutionStore
	Clock       service.Clock
}

var _ CancelNonterminalOperation = CancelNonterminal{}

// Execute is an inert stub until the access-ended cancellation behavior is implemented.
func (u CancelNonterminal) Execute(_ context.Context, _ CancelNonterminalInput) error {
	return errCancelNotImplemented
}
