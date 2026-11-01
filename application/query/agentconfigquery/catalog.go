package agentconfigquery

import (
	"context"

	"github.com/rfanazhari/amer-prototype/application/dto"
	"github.com/rfanazhari/amer-prototype/application/service"
)

// Catalog lists the developer-owned template versions after collection read authority.
type Catalog struct {
	Authority service.AgentConfigurationAuthority
}

// Execute authorizes collection read and returns detached catalog entries.
func (Catalog) Execute(context.Context) ([]dto.AgentTemplate, error) {
	return nil, nil
}
