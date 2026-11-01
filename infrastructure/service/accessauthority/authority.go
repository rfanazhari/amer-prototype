package accessauthority

import (
	"context"

	"github.com/rfanazhari/amer-prototype/application/service"
)

// Authority resolves a trusted administrative principal and current HR configuration authority.
type Authority struct {
	Sessions service.CurrentAdministrativeSession
	Clock    service.Clock
}

var _ service.AgentConfigurationAuthority = Authority{}

// Principal returns the stable account UUID after current session and account binding succeed.
func (a Authority) Principal(ctx context.Context) (service.AdminPrincipal, error) {
	session, account, err := a.Sessions.Resolve(ctx)
	if err != nil {
		return service.AdminPrincipal{}, err
	}
	if err := authorizeCurrent(session, account, trustedNow(a.Clock)); err != nil {
		return service.AdminPrincipal{}, err
	}
	return service.AdminPrincipal{SubjectID: account.ID}, nil
}

// Authorize rechecks the bound session and permits only current HR configuration operations.
func (a Authority) Authorize(ctx context.Context, principal service.AdminPrincipal, operation, resource string) error {
	session, account, err := a.Sessions.Resolve(ctx)
	if err != nil {
		return err
	}
	if principal.SubjectID == "" || principal.SubjectID != account.ID {
		return service.ErrSessionUnbound
	}
	if err := authorizeCurrent(session, account, trustedNow(a.Clock)); err != nil {
		return err
	}
	if !service.KnownAgentConfigurationOperation(operation) || !service.AllowsAgentConfiguration(account.Role) {
		return service.ErrForbidden
	}
	_ = resource
	return nil
}
