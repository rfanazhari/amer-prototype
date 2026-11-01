package accessusecase

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/rfanazhari/amer-prototype/application/dto"
	"github.com/rfanazhari/amer-prototype/application/repository/accessport"
	"github.com/rfanazhari/amer-prototype/application/service"
	"github.com/rfanazhari/amer-prototype/domain/aggregate/access"
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
	now := trustedNow(u.Clock)
	digest, err := u.Tokens.Digest(service.CredentialInvitation, token)
	if err != nil {
		return nil, err
	}
	record, err := u.Invitations.FindByTokenHash(ctx, digest)
	if errors.Is(err, accessport.ErrNotFound) {
		return nil, accessport.ErrInvitationUnavailable
	}
	if err != nil {
		return nil, err
	}
	invitation, err := restoreInvitation(record)
	if err != nil || !invitation.Current(now) {
		return nil, accessport.ErrInvitationUnavailable
	}
	var accessView dto.CandidateAccess
	err = u.Assignments.WithAssignment(ctx, invitation.AssignmentID(), func(tx accessport.AssignmentTx) error {
		facts, err := tx.Facts(ctx)
		if err != nil {
			return err
		}
		if !facts.PermitsExchange() || facts.AssignmentID != invitation.AssignmentID() {
			return accessport.ErrInvitationUnavailable
		}
		plaintext, sessionDigest, err := u.Tokens.Issue(ctx, service.CredentialCandidate)
		if err != nil {
			return err
		}
		session, err := access.NewCandidateSession(uuid.NewString(), facts.AssignmentID, record.ID, now)
		if err != nil {
			return err
		}
		snapshot := session.Snapshot()
		sessionRecord := accessport.CandidateSessionRecord{
			ID:           snapshot.ID,
			AssignmentID: snapshot.AssignmentID,
			InvitationID: snapshot.InvitationID,
			TokenHash:    sessionDigest,
			CreatedAt:    snapshot.CreatedAt,
			ExpiresAt:    snapshot.ExpiresAt,
			RevokedAt:    snapshot.RevokedAt,
		}
		if err := tx.InsertCandidateSession(ctx, sessionRecord); err != nil {
			return err
		}
		candidate, err := u.Candidates.FindByID(ctx, facts.CandidateID)
		if err != nil {
			return err
		}
		accessView = dto.CandidateAccess{
			Candidate:    dto.FromCandidate(candidate),
			AssignmentID: facts.AssignmentID,
			Session:      dto.AdministrativeSession{ID: sessionRecord.ID, ExpiresAt: sessionRecord.ExpiresAt},
			SessionToken: plaintext,
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &accessView, nil
}
