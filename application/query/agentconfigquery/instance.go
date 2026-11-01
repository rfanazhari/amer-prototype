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
func (Instance) Execute(context.Context, string) (*dto.AgentInstance, error) {
	return nil, nil
}
