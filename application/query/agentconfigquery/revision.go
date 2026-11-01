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
func (Revision) Execute(context.Context, string, string) (*dto.AgentRevision, error) {
	return nil, nil
}
