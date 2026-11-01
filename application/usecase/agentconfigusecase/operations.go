package agentconfigusecase

import (
	"context"

	"github.com/rfanazhari/amer-prototype/application/dto"
)

// CreateOperation is the create command contract consumed by delivery.
type CreateOperation interface {
	// Execute authorizes collection create and returns a detached instance only after commit.
	Execute(context.Context, CreateInput) (*dto.AgentInstance, error)
}

// ReplaceOperation is the draft-replacement command contract consumed by delivery.
type ReplaceOperation interface {
	// Execute authorizes replacement and returns a detached instance only after commit.
	Execute(context.Context, ReplaceInput) (*dto.AgentInstance, error)
}

// ValidateOperation is the exact-version validation command contract consumed by delivery.
type ValidateOperation interface {
	// Execute authorizes validation and records findings only for the locked expected draft.
	Execute(context.Context, ValidateInput) (*dto.AgentValidation, error)
}

// PublishResult is the detached publication outcome, including whether the command was replayed.
type PublishResult struct {
	Revision *dto.AgentRevision
	Replayed bool
}

// PublishOperation is the publication command contract consumed by delivery.
type PublishOperation interface {
	// Execute authorizes publication, checks replay before current-draft eligibility, and withholds results until commit.
	Execute(context.Context, PublishInput) (*PublishResult, error)
}
