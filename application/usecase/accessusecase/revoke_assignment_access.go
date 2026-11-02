package accessusecase

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/rfanazhari/amer-prototype/application/dto"
	"github.com/rfanazhari/amer-prototype/application/repository/accessport"
	"github.com/rfanazhari/amer-prototype/application/service"
	"github.com/rfanazhari/amer-prototype/application/usecase/executionusecase"
)

const (
	// accessRevokedReason is the terminal reason stored on a nonterminal execution cancelled by this command.
	accessRevokedReason = "access_revoked"
	// canonicalUUIDLength is the length of a hyphenated UUID, the only accepted assignment id form.
	canonicalUUIDLength = 36
)

// RevokeAssignmentAccess withdraws assignment access under current HR authority and cancels a nonterminal run.
type RevokeAssignmentAccess struct {
	Assignments accessport.AssignmentAccess
	Current     service.CurrentAdministrativeSession
	Lifecycle   executionusecase.CancelNonterminalOperation
	Clock       service.Clock
}

var _ RevokeAssignmentAccessOperation = RevokeAssignmentAccess{}

// Execute commits access_revoked_at while it is still null, then asks the lifecycle to cancel a nonterminal
// execution. The lifecycle call runs after every successful access commit, including a repeat, and its failure
// is reported as an incomplete cancellation without a success result.
func (u RevokeAssignmentAccess) Execute(ctx context.Context, assignmentID string) (*dto.AssignmentRevocation, error) {
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
	if _, err := uuid.Parse(assignmentID); err != nil || len(assignmentID) != canonicalUUIDLength {
		return nil, service.ErrInvalidInput
	}
	revokedAt, err := u.commitMarker(ctx, assignmentID, now)
	if err != nil {
		return nil, err
	}
	input := executionusecase.CancelNonterminalInput{AssignmentID: assignmentID, ActorRef: actor.ID, Reason: accessRevokedReason}
	if err := u.Lifecycle.Execute(ctx, input); err != nil {
		return nil, fmt.Errorf("%w: %w", service.ErrCancellationIncomplete, err)
	}
	return &dto.AssignmentRevocation{ID: assignmentID, AccessRevokedAt: revokedAt}, nil
}

// commitMarker locks the assignment and returns the stored revocation instant, writing now only while it is null.
func (u RevokeAssignmentAccess) commitMarker(ctx context.Context, assignmentID string, now time.Time) (time.Time, error) {
	revokedAt := now
	err := u.Assignments.WithAssignment(ctx, assignmentID, func(tx accessport.AssignmentTx) error {
		facts, err := tx.Facts(ctx)
		if err != nil {
			return err
		}
		if facts.AccessRevokedAt != nil {
			revokedAt = facts.AccessRevokedAt.UTC()
			return nil
		}
		return tx.MarkAccessRevoked(ctx, now)
	})
	if err != nil {
		return time.Time{}, err
	}
	return revokedAt, nil
}
