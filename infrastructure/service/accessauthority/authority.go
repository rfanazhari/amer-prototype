package accessauthority

import (
	"context"

	"github.com/rfanazhari/amer-prototype/application/service"
	"github.com/rfanazhari/amer-prototype/domain/enum"
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
	_ = resource
	return a.authorize(ctx, principal, service.KnownAgentConfigurationOperation(operation), service.AllowsAgentConfiguration)
}

// authorize rechecks the bound session, then applies the supplied operation and role decisions.
func (a Authority) authorize(ctx context.Context, principal service.AdminPrincipal, knownOperation bool, allows func(enum.Role) bool) error {
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
	if !knownOperation || !allows(account.Role) {
		return service.ErrForbidden
	}
	return nil
}
