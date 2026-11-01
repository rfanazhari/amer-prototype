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

// CreateCandidateOperation records a reusable candidate identity under current HR authority.
type CreateCandidateOperation interface {
	// Execute persists a new candidate after current HR recheck.
	Execute(ctx context.Context, name, email string) (*dto.Candidate, error)
}

// IssueInvitationOperation issues or replaces an assignment invitation under current HR authority.
type IssueInvitationOperation interface {
	// Execute persists a new invitation after assignment-first eligibility checks and returns the plaintext token once.
	Execute(ctx context.Context, assignmentID string) (*dto.IssuedInvitation, error)
}

// ExchangeInvitationOperation establishes a candidate session from an invitation secret.
type ExchangeInvitationOperation interface {
	// Execute accepts only the invitation token and returns session metadata after commit.
	Execute(ctx context.Context, token string) (*dto.CandidateAccess, error)
}
