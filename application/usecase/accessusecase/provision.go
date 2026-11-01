package accessusecase

import (
	"context"

	"github.com/rfanazhari/amer-prototype/application/dto"
	"github.com/rfanazhari/amer-prototype/application/repository/accessport"
	"github.com/rfanazhari/amer-prototype/application/service"
)

// ProvisionHR creates an HR account under the current ADMIN authority.
type ProvisionHR struct {
	Accounts accessport.Accounts
	Current  service.CurrentAdministrativeSession
	Hasher   service.PasswordHasher
	Clock    service.Clock
}

// Execute is an inert T6 RED stub; GREEN persists HR only after current ADMIN recheck.
func (ProvisionHR) Execute(context.Context, string, string, string) (*dto.AdministrativeAccount, error) {
	return nil, nil
}
