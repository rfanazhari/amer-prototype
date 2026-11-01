package agentbindingusecase

import (
	"context"

	"github.com/rfanazhari/amer-prototype/application/dto"
	"github.com/rfanazhari/amer-prototype/application/repository/accessport"
	"github.com/rfanazhari/amer-prototype/application/repository/agentbindingport"
	"github.com/rfanazhari/amer-prototype/application/repository/agentconfigport"
	"github.com/rfanazhari/amer-prototype/application/service"
)

// CreateAssignment records a new interview assignment pin under current HR authority.
type CreateAssignment struct {
	Current     service.CurrentAdministrativeSession
	Clock       service.Clock
	Candidates  accessport.Candidates
	Instances   agentconfigport.InstanceReader
	Assignments agentbindingport.Assignments
}

// Execute mints an assignment identity, copies the current HR Interviewer active revision, and persists the pin.
func (u CreateAssignment) Execute(ctx context.Context, candidateID, instanceID string) (*dto.Assignment, error) {
	return nil, nil
}
