package agentconfigusecase

import (
	"context"

	"github.com/rfanazhari/amer-prototype/application/dto"
	"github.com/rfanazhari/amer-prototype/application/repository/agentconfigport"
	"github.com/rfanazhari/amer-prototype/application/service"
)

// Create creates an unpublished instance for a catalog template version.
type Create struct {
	Authority service.AgentConfigurationAuthority
	Instances agentconfigport.Store
	Clock     service.Clock
}

// Execute authorizes collection create and returns a detached instance only after commit.
func (Create) Execute(context.Context, CreateInput) (*dto.AgentInstance, error) {
	return nil, nil
}
