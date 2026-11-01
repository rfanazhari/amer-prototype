package agentbindingusecase

import (
	"context"
	"errors"

	"github.com/rfanazhari/amer-prototype/application/repository/agentbindingport"
	"github.com/rfanazhari/amer-prototype/application/repository/agentconfigport"
)

// PinAssignment binds a new assignment to the current active published revision.
type PinAssignment struct {
	Instances   agentconfigport.InstanceReader
	Assignments agentbindingport.Assignments
}

// Execute is an inert RED stub; GREEN selects and persists the active revision pin.
func (u PinAssignment) Execute(ctx context.Context, assignmentID, candidateID, instanceID string) (agentbindingport.AssignmentBinding, error) {
	_ = ctx
	_ = assignmentID
	_ = candidateID
	_ = instanceID
	_ = u.Instances
	_ = u.Assignments
	return agentbindingport.AssignmentBinding{}, errors.New("pin assignment is not implemented")
}
