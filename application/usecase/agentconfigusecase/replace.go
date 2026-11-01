package agentconfigusecase

import (
	"context"

	"github.com/rfanazhari/amer-prototype/application/dto"
	"github.com/rfanazhari/amer-prototype/application/repository/agentconfigport"
	"github.com/rfanazhari/amer-prototype/application/service"
)

// Replace performs an exact-version full draft replacement.
type Replace struct {
	Authority service.AgentConfigurationAuthority
	Instances agentconfigport.Store
	Clock     service.Clock
}

// Execute authorizes replacement and returns a detached instance only after commit.
func (Replace) Execute(context.Context, ReplaceInput) (*dto.AgentInstance, error) {
	return nil, nil
}
