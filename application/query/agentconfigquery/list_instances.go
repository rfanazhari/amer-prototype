package agentconfigquery

import (
	"context"

	"github.com/rfanazhari/amer-prototype/application/dto"
	"github.com/rfanazhari/amer-prototype/application/repository/agentconfigport"
	"github.com/rfanazhari/amer-prototype/application/service"
)

// ListInstances returns every authorized detached instance view.
type ListInstances struct {
	Authority service.AgentConfigurationAuthority
	Lister    agentconfigport.InstanceLister
}

// Execute authorizes one collection read, then returns every filtered instance view.
func (q ListInstances) Execute(ctx context.Context) ([]dto.AgentInstance, error) {
	if _, err := authorize(ctx, q.Authority, service.AgentConfigurationRead, ""); err != nil {
		return nil, err
	}
	stored, err := q.Lister.ListInstances(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]dto.AgentInstance, 0, len(stored))
	for _, item := range stored {
		view, err := dto.FromStoredInstance(item)
		if err != nil {
			return nil, err
		}
		out = append(out, *view)
	}
	return out, nil
}
