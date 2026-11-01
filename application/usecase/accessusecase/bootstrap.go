package accessusecase

import (
	"context"

	"github.com/rfanazhari/amer-prototype/application/repository/accessport"
	"github.com/rfanazhari/amer-prototype/application/service"
)

// Bootstrap initializes the first ADMIN when applicable without overwriting existing accounts.
type Bootstrap struct {
	Accounts accessport.Accounts
	Hasher   service.PasswordHasher
	Clock    service.Clock
	Name     string
	Email    string
	Password string
}

// Execute is an inert T6 RED stub; GREEN coordinates hashing and serialized initialization.
func (Bootstrap) Execute(context.Context) error { return nil }
