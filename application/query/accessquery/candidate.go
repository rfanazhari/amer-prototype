package accessquery

import (
	"context"
	"errors"

	"github.com/rfanazhari/amer-prototype/application/dto"
	"github.com/rfanazhari/amer-prototype/application/repository/accessport"
	"github.com/rfanazhari/amer-prototype/application/service"
)

// Candidate returns a detached candidate identity for current HR selection.
type Candidate struct {
	Candidates accessport.Candidates
	Current    service.CurrentAdministrativeSession
	Clock      service.Clock
}

// Execute returns one candidate by identity after current HR authorization.
func (q Candidate) Execute(ctx context.Context, id string) (*dto.Candidate, error) {
	return nil, errors.New("read candidate is not implemented")
}
