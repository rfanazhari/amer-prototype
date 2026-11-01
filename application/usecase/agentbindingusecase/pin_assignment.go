package agentbindingusecase

import (
	"context"

	"github.com/rfanazhari/amer-prototype/application/repository/agentbindingport"
	"github.com/rfanazhari/amer-prototype/application/repository/agentconfigport"
)

// PinAssignment binds a new assignment to the current active published revision.
type PinAssignment struct {
	Instances   agentconfigport.InstanceReader
	Assignments agentbindingport.Assignments
}

// Execute stores the assignment pin selected from the current active revision.
func (u PinAssignment) Execute(ctx context.Context, assignmentID, candidateID, instanceID string) (agentbindingport.AssignmentBinding, error) {
	stored, err := u.Instances.FindInstance(ctx, instanceID)
	if err != nil {
		return agentbindingport.AssignmentBinding{}, err
	}
	revisionID, err := activeRevisionID(stored)
	if err != nil {
		return agentbindingport.AssignmentBinding{}, err
	}
	binding := agentbindingport.AssignmentBinding{
		AssignmentID: assignmentID, CandidateID: candidateID, InstanceID: instanceID, RevisionID: revisionID,
	}
	if err := u.Assignments.Insert(ctx, binding); err != nil {
		return agentbindingport.AssignmentBinding{}, err
	}
	return binding, nil
}

// activeRevisionID returns the instance-owned active published revision identity.
func activeRevisionID(stored agentconfigport.StoredInstance) (string, error) {
	switch {
	case stored.HR != nil:
		active := stored.HR.Snapshot().Active
		if active == nil {
			return "", agentbindingport.ErrNoActiveRevision
		}
		return active.ID, nil
	case stored.Document != nil:
		active := stored.Document.Snapshot().Active
		if active == nil {
			return "", agentbindingport.ErrNoActiveRevision
		}
		return active.ID, nil
	default:
		return "", agentconfigport.ErrNotFound
	}
}
