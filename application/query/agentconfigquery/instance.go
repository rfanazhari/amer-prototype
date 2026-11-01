package agentconfigquery

import (
	"context"

	"github.com/rfanazhari/amer-prototype/application/dto"
	"github.com/rfanazhari/amer-prototype/application/repository/agentconfigport"
	"github.com/rfanazhari/amer-prototype/application/service"
)

// Instance returns one authorized detached instance view.
type Instance struct {
	Authority service.AgentConfigurationAuthority
	Reader    agentconfigport.InstanceReader
}

// Execute authorizes read before disclosing a stored instance.
func (q Instance) Execute(ctx context.Context, id string) (*dto.AgentInstance, error) {
	if _, err := authorize(ctx, q.Authority, service.AgentConfigurationRead, id); err != nil {
		return nil, err
	}
	stored, err := q.Reader.FindInstance(ctx, id)
	if err != nil {
		return nil, err
	}
	return dto.FromStoredInstance(stored)
}
