package executionrepository

import (
	"context"
	"errors"

	"github.com/rfanazhari/amer-prototype/application/repository/executionport"
	"github.com/rfanazhari/amer-prototype/domain/aggregate/execution"
	"gorm.io/gorm"
)

// errNotImplemented marks behavior that the store has not provided yet.
var errNotImplemented = errors.New("execution store behavior not implemented")

// Store persists executions and the assignment lock through a shared GORM database pool.
type Store struct{ DB *gorm.DB }

var (
	_ executionport.AssignmentLock = Store{}
	_ executionport.ExecutionStore = Store{}
)

// WithAssignment locks the assignment row and runs fn inside the transaction.
func (s Store) WithAssignment(_ context.Context, _ string, _ func(context.Context, executionport.AssignmentRow) error) error {
	return errNotImplemented
}

// Insert writes a CREATED execution and its initial transition.
func (s Store) Insert(_ context.Context, _ execution.Snapshot, _ execution.TransitionFact) error {
	return errNotImplemented
}

// InsertInterview writes the execution, its initial transition, and then the assignment pin.
func (s Store) InsertInterview(_ context.Context, _ string, _ execution.Snapshot, _ execution.TransitionFact) error {
	return errNotImplemented
}

// Update locks the execution row and persists the callback's change.
func (s Store) Update(_ context.Context, _ string, _ func(*execution.Snapshot) (*execution.TransitionFact, error)) error {
	return errNotImplemented
}

// Find returns the stored execution snapshot by id.
func (s Store) Find(_ context.Context, _ string) (execution.Snapshot, error) {
	return execution.Snapshot{}, errNotImplemented
}

// FindByAssignment returns the execution linked to the assignment by the interview pin.
func (s Store) FindByAssignment(_ context.Context, _ string) (execution.Snapshot, error) {
	return execution.Snapshot{}, errNotImplemented
}
