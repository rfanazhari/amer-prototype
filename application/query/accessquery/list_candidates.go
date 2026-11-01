package accessquery

import (
	"context"

	"github.com/rfanazhari/amer-prototype/application/dto"
	"github.com/rfanazhari/amer-prototype/application/repository/accessport"
	"github.com/rfanazhari/amer-prototype/application/service"
)

// ListCandidates returns candidates created by the logged-in HR account.
type ListCandidates struct {
	Candidates accessport.Candidates
	Current    service.CurrentAdministrativeSession
	Clock      service.Clock
}

// Execute returns that account's candidates after current HR authorization.
func (q ListCandidates) Execute(ctx context.Context) ([]dto.Candidate, error) {
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
	records, err := q.Candidates.ListByCreator(ctx, actor.ID)
	if err != nil {
		return nil, err
	}
	views := make([]dto.Candidate, 0, len(records))
	for _, record := range records {
		views = append(views, dto.FromCandidate(record))
	}
	return views, nil
}
