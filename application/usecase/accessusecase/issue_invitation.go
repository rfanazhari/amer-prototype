package accessusecase

import (
	"context"

	"github.com/google/uuid"
	"github.com/rfanazhari/amer-prototype/application/dto"
	"github.com/rfanazhari/amer-prototype/application/repository/accessport"
	"github.com/rfanazhari/amer-prototype/application/service"
	"github.com/rfanazhari/amer-prototype/domain/aggregate/access"
)

// IssueInvitation issues or replaces an assignment invitation under current HR authority.
type IssueInvitation struct {
	Assignments accessport.AssignmentAccess
	Current     service.CurrentAdministrativeSession
	Tokens      service.TokenSecrets
	Clock       service.Clock
}

// Execute persists a new invitation after assignment-first eligibility checks and returns the plaintext token once.
func (u IssueInvitation) Execute(ctx context.Context, assignmentID string) (*dto.IssuedInvitation, error) {
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
	var issued dto.IssuedInvitation
	err = u.Assignments.WithAssignment(ctx, assignmentID, func(tx accessport.AssignmentTx) error {
		facts, err := tx.Facts(ctx)
		if err != nil {
			return err
		}
		if !facts.PermitsInvitation(facts.CandidateID) {
			return service.ErrForbidden
		}
		plaintext, digest, err := u.Tokens.Issue(ctx, service.CredentialInvitation)
		if err != nil {
			return err
		}
		invitation, err := access.NewInvitation(uuid.NewString(), facts.AssignmentID, actor.ID, now)
		if err != nil {
			return err
		}
		current, err := tx.CurrentInvitations(ctx)
		if err != nil {
			return err
		}
		for _, existing := range current {
			restored, err := restoreInvitation(existing)
			if err != nil {
				return err
			}
			if err := restored.Revoke(now); err != nil {
				return err
			}
			revoked := restored.Snapshot()
			if err := tx.RevokeInvitation(ctx, existing.ID, revoked.RevokedAt); err != nil {
				return err
			}
		}
		snapshot := invitation.Snapshot()
		record := accessport.InvitationRecord{
			ID:           snapshot.ID,
			AssignmentID: snapshot.AssignmentID,
			TokenHash:    digest,
			CreatedBy:    snapshot.CreatedBy,
			CreatedAt:    snapshot.CreatedAt,
			ExpiresAt:    snapshot.ExpiresAt,
			RevokedAt:    snapshot.RevokedAt,
		}
		if err := tx.InsertInvitation(ctx, record); err != nil {
			return err
		}
		issued = dto.IssuedInvitation{
			ID:           record.ID,
			AssignmentID: record.AssignmentID,
			ExpiresAt:    record.ExpiresAt,
			Token:        plaintext,
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &issued, nil
}

// restoreInvitation rebuilds the invitation entity from a detached record.
func restoreInvitation(record accessport.InvitationRecord) (*access.Invitation, error) {
	return access.RestoreInvitation(access.InvitationSnapshot{
		ID:           record.ID,
		AssignmentID: record.AssignmentID,
		CreatedBy:    record.CreatedBy,
		CreatedAt:    record.CreatedAt,
		ExpiresAt:    record.ExpiresAt,
		RevokedAt:    record.RevokedAt,
	})
}
