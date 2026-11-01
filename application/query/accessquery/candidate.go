package accessquery

import (
	"context"

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
	session, actor, err := q.Current.Resolve(ctx)
	if err != nil {
		return nil, err
	}
	now := trustedQueryNow(q.Clock)
	if err := authorizeAdministrative(session, actor, now); err != nil {
		return nil, err
	}
	if !service.AllowsCandidateAdministration(actor.Role) {
		return nil, service.ErrForbidden
	}
	record, err := q.Candidates.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	view := dto.FromCandidate(record)
	return &view, nil
}
