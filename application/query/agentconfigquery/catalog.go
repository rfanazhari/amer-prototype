package agentconfigquery

import (
	"context"

	"github.com/rfanazhari/amer-prototype/application/dto"
	"github.com/rfanazhari/amer-prototype/application/service"
	catalog "github.com/rfanazhari/amer-prototype/domain/agentconfig"
)

// Catalog lists the developer-owned template versions after collection read authority.
type Catalog struct {
	Authority service.AgentConfigurationAuthority
}

// Execute authorizes collection read and returns detached catalog entries.
func (q Catalog) Execute(ctx context.Context) ([]dto.AgentTemplate, error) {
	if _, err := authorize(ctx, q.Authority, service.AgentConfigurationRead, ""); err != nil {
		return nil, err
	}
	return dto.FromCatalog(catalog.Templates()), nil
}
