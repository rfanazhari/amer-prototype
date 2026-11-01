package accessquery

import (
	"context"

	"github.com/rfanazhari/amer-prototype/application/dto"
	"github.com/rfanazhari/amer-prototype/application/repository/accessport"
	"github.com/rfanazhari/amer-prototype/application/service"
	"github.com/rfanazhari/amer-prototype/domain/aggregate/access"
)

// CandidateSession returns the current candidate session after trusted assignment binding.
type CandidateSession struct {
	Current     service.CurrentCandidateSession
	Candidates  accessport.Candidates
	Assignments accessport.AssignmentAccess
	CSRF        service.CSRFTokens
	Clock       service.Clock
}

// Execute returns detached candidate, assignment and session material without creating execution.
func (q CandidateSession) Execute(ctx context.Context) (*dto.CandidateAccess, error) {
	record, err := q.Current.Resolve(ctx)
	if err != nil {
		return nil, err
	}
	now := trustedQueryNow(q.Clock)
	session, err := access.RestoreCandidateSession(access.CandidateSessionSnapshot{
		ID:           record.ID,
		AssignmentID: record.AssignmentID,
		InvitationID: record.InvitationID,
		CreatedAt:    record.CreatedAt,
		ExpiresAt:    record.ExpiresAt,
		RevokedAt:    record.RevokedAt,
	})
	if err != nil || !session.Current(now) {
		return nil, service.ErrCandidateSessionUnbound
	}
	var view dto.CandidateAccess
	err = q.Assignments.WithAssignment(ctx, record.AssignmentID, func(tx accessport.AssignmentTx) error {
		facts, err := tx.Facts(ctx)
		if err != nil {
			return err
		}
		if !facts.PermitsSessionAccess() || !session.Authorizes(facts.AssignmentID, now) {
			return service.ErrCandidateSessionUnbound
		}
		candidate, err := q.Candidates.FindByID(ctx, facts.CandidateID)
		if err != nil {
			return err
		}
		token, err := q.CSRF.Issue(service.CredentialCandidate, record.ID)
		if err != nil {
			return err
		}
		view = dto.CandidateAccess{
			Candidate:    dto.FromCandidate(candidate),
			AssignmentID: facts.AssignmentID,
			Session:      dto.AdministrativeSession{ID: record.ID, ExpiresAt: record.ExpiresAt},
			CSRFToken:    token,
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &view, nil
}
