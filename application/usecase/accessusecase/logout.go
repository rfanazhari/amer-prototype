package accessusecase

import (
	"context"

	"github.com/rfanazhari/amer-prototype/application/repository/accessport"
	"github.com/rfanazhari/amer-prototype/application/service"
)

// Logout persists current-session revocation before cookie clearing.
type Logout struct {
	Current  service.CurrentAdministrativeSession
	Sessions accessport.Sessions
	Clock    service.Clock
}

// Execute is an inert T6 RED stub; GREEN revokes only after current binding succeeds.
func (Logout) Execute(context.Context) error { return nil }
