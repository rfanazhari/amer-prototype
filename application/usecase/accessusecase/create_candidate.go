package accessusecase

import (
	"context"
	"errors"

	"github.com/rfanazhari/amer-prototype/application/dto"
	"github.com/rfanazhari/amer-prototype/application/repository/accessport"
	"github.com/rfanazhari/amer-prototype/application/service"
)

// CreateCandidate records a reusable candidate identity under current HR authority.
type CreateCandidate struct {
	Candidates accessport.Candidates
	Current    service.CurrentAdministrativeSession
	Clock      service.Clock
}

// Execute persists a new candidate after current HR recheck.
func (u CreateCandidate) Execute(ctx context.Context, name, email string) (*dto.Candidate, error) {
	return nil, errors.New("create candidate is not implemented")
}
