package agentbindingrepository

import (
	"context"
	"errors"

	"github.com/rfanazhari/amer-prototype/application/repository/agentbindingport"
	"gorm.io/gorm"
)

// Assignments persists Interview-shaped assignment revision pins.
type Assignments struct {
	DB *gorm.DB
}

// Executions persists Interview-shaped execution revision pins.
type Executions struct {
	DB *gorm.DB
}

// Insert is an inert RED stub; GREEN writes interview_assignments.
func (s Assignments) Insert(ctx context.Context, binding agentbindingport.AssignmentBinding) error {
	_ = ctx
	_ = binding
	_ = s.DB
	return errors.New("assignment binding insert is not implemented")
}

// Find is an inert RED stub; GREEN restores one assignment pin.
func (s Assignments) Find(ctx context.Context, assignmentID string) (agentbindingport.AssignmentBinding, error) {
	_ = ctx
	_ = assignmentID
	_ = s.DB
	return agentbindingport.AssignmentBinding{}, errors.New("assignment binding find is not implemented")
}

// Insert is an inert RED stub; GREEN writes interview_executions.
func (s Executions) Insert(ctx context.Context, binding agentbindingport.ExecutionBinding) error {
	_ = ctx
	_ = binding
	_ = s.DB
	return errors.New("execution binding insert is not implemented")
}

// Find is an inert RED stub; GREEN restores one execution pin.
func (s Executions) Find(ctx context.Context, executionID string) (agentbindingport.ExecutionBinding, error) {
	_ = ctx
	_ = executionID
	_ = s.DB
	return agentbindingport.ExecutionBinding{}, errors.New("execution binding find is not implemented")
}
