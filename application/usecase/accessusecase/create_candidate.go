package accessusecase

import (
	"context"

	"github.com/google/uuid"
	"github.com/rfanazhari/amer-prototype/application/dto"
	"github.com/rfanazhari/amer-prototype/application/repository/accessport"
	"github.com/rfanazhari/amer-prototype/application/service"
	"github.com/rfanazhari/amer-prototype/domain/aggregate/access"
)

// CreateCandidate records a reusable candidate identity under current HR authority.
type CreateCandidate struct {
	Candidates accessport.Candidates
	Current    service.CurrentAdministrativeSession
	Clock      service.Clock
}

// Execute persists a new candidate after current HR recheck.
func (u CreateCandidate) Execute(ctx context.Context, name, email string) (*dto.Candidate, error) {
	session, actor, err := u.Current.Resolve(ctx)
	if err != nil {
		return nil, err
	}
	now := trustedNow(u.Clock)
	if err := authorizeCurrent(session, actor, now); err != nil {
		return nil, err
	}
	if !service.AllowsCandidateAdministration(actor.Role) {
		return nil, service.ErrForbidden
	}
	canonicalName, err := service.CanonicalAdministrativeName(name)
	if err != nil {
		return nil, err
	}
	canonicalEmail, err := service.CanonicalAdministrativeEmail(email)
	if err != nil {
		return nil, err
	}
	candidate, err := access.NewCandidate(uuid.NewString(), canonicalName, canonicalEmail, actor.ID, now)
	if err != nil {
		return nil, err
	}
	snapshot := candidate.Snapshot()
	record := accessport.CandidateRecord{
		ID:        snapshot.ID,
		Name:      snapshot.Name,
		Email:     snapshot.Email,
		CreatedBy: snapshot.CreatedBy,
		CreatedAt: snapshot.CreatedAt,
		UpdatedAt: snapshot.UpdatedAt,
	}
	if err := u.Candidates.Insert(ctx, record); err != nil {
		return nil, err
	}
	view := dto.FromCandidate(record)
	return &view, nil
}
