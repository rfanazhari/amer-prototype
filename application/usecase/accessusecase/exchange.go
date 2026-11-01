package accessusecase

import (
	"context"
	"errors"

	"github.com/rfanazhari/amer-prototype/application/dto"
	"github.com/rfanazhari/amer-prototype/application/repository/accessport"
	"github.com/rfanazhari/amer-prototype/application/service"
)

// ExchangeInvitation establishes a candidate session from an invitation secret.
type ExchangeInvitation struct {
	Invitations accessport.Invitations
	Candidates  accessport.Candidates
	Assignments accessport.AssignmentAccess
	Tokens      service.TokenSecrets
	Clock       service.Clock
}

// Execute accepts only the invitation token and returns session metadata after commit.
func (u ExchangeInvitation) Execute(ctx context.Context, token string) (*dto.CandidateAccess, error) {
	return nil, errors.New("exchange invitation is not implemented")
}
