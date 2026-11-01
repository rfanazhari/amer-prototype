package agentconfigquery

import (
	"context"

	"github.com/rfanazhari/amer-prototype/application/service"
)

// authorize resolves the trusted principal and checks current authority before any disclosure.
func authorize(ctx context.Context, authority service.AgentConfigurationAuthority, operation, resource string) (service.AdminPrincipal, error) {
	principal, err := authority.Principal(ctx)
	if err != nil {
		return service.AdminPrincipal{}, err
	}
	if err := authority.Authorize(ctx, principal, operation, resource); err != nil {
		return service.AdminPrincipal{}, err
	}
	return principal, nil
}
