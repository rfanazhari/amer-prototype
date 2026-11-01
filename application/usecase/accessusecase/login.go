package accessusecase

import (
	"context"

	"github.com/rfanazhari/amer-prototype/application/dto"
	"github.com/rfanazhari/amer-prototype/application/repository/accessport"
	"github.com/rfanazhari/amer-prototype/application/service"
)

// Login establishes an administrative session after credential proof and commit.
type Login struct {
	Accounts accessport.Accounts
	Issuance accessport.SessionIssuance
	Hasher   service.PasswordHasher
	Tokens   service.TokenSecrets
	Clock    service.Clock
}

// Execute is an inert T6 RED stub; GREEN returns metadata only after session commit.
func (Login) Execute(context.Context, string, string) (*dto.AdministrativeLogin, error) {
	return nil, nil
}
