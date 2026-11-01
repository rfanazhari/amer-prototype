package agentconfigquery

import (
	"context"

	"github.com/rfanazhari/amer-prototype/application/dto"
	"github.com/rfanazhari/amer-prototype/application/repository/agentconfigport"
	"github.com/rfanazhari/amer-prototype/application/service"
)

// Revision returns one authorized detached immutable revision view.
type Revision struct {
	Authority service.AgentConfigurationAuthority
	Reader    agentconfigport.InstanceReader
}

// Execute authorizes read before disclosing a stored revision.
func (q Revision) Execute(ctx context.Context, instanceID, revisionID string) (*dto.AgentRevision, error) {
	if _, err := authorize(ctx, q.Authority, service.AgentConfigurationRead, instanceID); err != nil {
		return nil, err
	}
	stored, err := q.Reader.FindRevision(ctx, instanceID, revisionID)
	if err != nil {
		return nil, err
	}
	return dto.FromStoredRevision(stored)
}
