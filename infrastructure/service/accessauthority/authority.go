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

// Principal is an inert T7 RED stub; GREEN returns the stable account SubjectID.
func (Authority) Principal(context.Context) (service.AdminPrincipal, error) {
	return service.AdminPrincipal{}, nil
}

// Authorize is an inert T7 RED stub; GREEN rechecks current HR authority before configuration work.
func (Authority) Authorize(context.Context, service.AdminPrincipal, string, string) error {
	return nil
}
