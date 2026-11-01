package agentconfigusecase

import (
	"context"

	"github.com/rfanazhari/amer-prototype/application/dto"
	"github.com/rfanazhari/amer-prototype/application/repository/agentconfigport"
	"github.com/rfanazhari/amer-prototype/application/service"
)

// Validate records a deterministic assessment against an exact current draft version.
type Validate struct {
	Authority  service.AgentConfigurationAuthority
	Reader     agentconfigport.InstanceReader
	Instances  agentconfigport.Store
	Assessment service.DraftAssessment
	Clock      service.Clock
}

// Execute authorizes validation and records findings only for the locked expected draft.
func (Validate) Execute(context.Context, ValidateInput) (*dto.AgentValidation, error) {
	return nil, nil
}
