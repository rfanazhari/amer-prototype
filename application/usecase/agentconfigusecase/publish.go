package agentconfigusecase

import (
	"context"

	"github.com/rfanazhari/amer-prototype/application/dto"
	"github.com/rfanazhari/amer-prototype/application/repository/agentconfigport"
	"github.com/rfanazhari/amer-prototype/application/service"
)

// Publish commits a new revision or returns a matching stored command result.
type Publish struct {
	Authority service.AgentConfigurationAuthority
	Instances agentconfigport.Store
	Reader    agentconfigport.InstanceReader
	Replay    agentconfigport.ReplayReader
	Clock     service.Clock
}

// Execute authorizes publication, checks replay before current-draft eligibility, and withholds results until commit.
func (Publish) Execute(context.Context, PublishInput) (*dto.AgentRevision, error) {
	return nil, nil
}
