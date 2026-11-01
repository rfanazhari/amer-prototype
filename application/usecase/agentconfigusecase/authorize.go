package agentconfigusecase

import (
	"context"
	"time"

	"github.com/rfanazhari/amer-prototype/application/service"
)

// authorize resolves the trusted principal and checks current authority before any disclosure or write.
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

// trustedNow canonicalizes injected clock samples to UTC milliseconds.
func trustedNow(clock service.Clock) time.Time {
	return clock.Now().UTC().Truncate(time.Millisecond)
}
