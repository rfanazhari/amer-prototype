package accessusecase

import (
	"context"

	"github.com/rfanazhari/amer-prototype/application/dto"
)

// BootstrapOperation initializes the first ADMIN when applicable without overwriting existing accounts.
type BootstrapOperation interface {
	// Execute runs bounded bootstrap and preserves classified persistence failures.
	Execute(ctx context.Context) error
}

// ProvisionHROperation creates an HR account under the current ADMIN authority.
type ProvisionHROperation interface {
	// Execute persists a new HR identity after current ADMIN recheck and unique-email commit.
	Execute(ctx context.Context, name, email, password string) (*dto.AdministrativeAccount, error)
}

// LoginOperation establishes an administrative session after credential proof and commit.
type LoginOperation interface {
	// Execute returns detached account and session metadata only after session commit.
	Execute(ctx context.Context, email, password string) (*dto.AdministrativeLogin, error)
}

// LogoutOperation persists current-session revocation before cookie clearing.
type LogoutOperation interface {
	// Execute revokes the current administrative session or reports commit uncertainty.
	Execute(ctx context.Context) error
}
