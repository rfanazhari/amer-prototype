package agentbindingusecase

import (
	"context"
	"errors"

	"github.com/rfanazhari/amer-prototype/application/repository/agentbindingport"
	"github.com/rfanazhari/amer-prototype/application/repository/agentconfigport"
)

// LoadRequirements restores immutable published requirements from an execution pin.
type LoadRequirements struct {
	Executions agentbindingport.Executions
	Revisions  agentconfigport.InstanceReader
}

// Execute is an inert RED stub; GREEN returns the pinned revision's stored fields.
func (u LoadRequirements) Execute(ctx context.Context, executionID string) (Requirements, error) {
	_ = ctx
	_ = executionID
	_ = u.Executions
	_ = u.Revisions
	return Requirements{}, errors.New("load requirements is not implemented")
}
