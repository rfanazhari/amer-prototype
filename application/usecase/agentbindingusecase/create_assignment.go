package agentbindingusecase

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/rfanazhari/amer-prototype/application/dto"
	"github.com/rfanazhari/amer-prototype/application/repository/accessport"
	"github.com/rfanazhari/amer-prototype/application/repository/agentbindingport"
	"github.com/rfanazhari/amer-prototype/application/repository/agentconfigport"
	"github.com/rfanazhari/amer-prototype/application/service"
	catalog "github.com/rfanazhari/amer-prototype/domain/agentconfig"
	"github.com/rfanazhari/amer-prototype/domain/aggregate/access"
)

// CreateAssignment records a new interview assignment pin under current HR authority.
// Only an hr-interviewer instance with an active published revision may be pinned.
type CreateAssignment struct {
	Current     service.CurrentAdministrativeSession
	Clock       service.Clock
	Candidates  accessport.Candidates
	Instances   agentconfigport.InstanceReader
	Assignments agentbindingport.Assignments
}

// Execute mints an assignment identity, copies the current HR Interviewer active revision, and persists the pin.
func (u CreateAssignment) Execute(ctx context.Context, candidateID, instanceID string) (*dto.Assignment, error) {
	session, actor, err := u.Current.Resolve(ctx)
	if err != nil {
		return nil, err
	}
	now := trustedClock(u.Clock)
	if err := authorizeAssignmentActor(session, actor, now); err != nil {
		return nil, err
	}
	if !service.AllowsCandidateAdministration(actor.Role) {
		return nil, service.ErrForbidden
	}
	if _, err := u.Candidates.FindByID(ctx, candidateID); err != nil {
		return nil, err
	}
	stored, err := u.Instances.FindInstance(ctx, instanceID)
	if err != nil {
		return nil, err
	}
	if stored.Document != nil || (stored.HR != nil && stored.HR.Snapshot().Template.Template != catalog.TemplateHRInterviewer) {
		return nil, agentbindingport.ErrUnsupportedTemplate
	}
	revisionID, err := activeRevisionID(stored)
	if err != nil {
		return nil, err
	}
	binding := agentbindingport.AssignmentBinding{
		AssignmentID: uuid.NewString(), CandidateID: candidateID, InstanceID: instanceID, RevisionID: revisionID,
	}
	if err := u.Assignments.Insert(ctx, binding); err != nil {
		return nil, err
	}
	return &dto.Assignment{
		ID: binding.AssignmentID, CandidateID: binding.CandidateID, InstanceID: binding.InstanceID, PublishedRevisionID: binding.RevisionID,
	}, nil
}

// authorizeAssignmentActor rejects expired, revoked, disabled or mismatched session and account pairs.
func authorizeAssignmentActor(session accessport.SessionRecord, account accessport.AccountRecord, now time.Time) error {
	restoredAccount, err := access.RestoreAdministrativeAccount(access.AccountSnapshot{
		ID: account.ID, Role: account.Role, Status: account.Status, CreatedAt: account.CreatedAt, UpdatedAt: account.UpdatedAt,
	})
	if err != nil {
		return err
	}
	restoredSession, err := access.RestoreAdministrativeSession(access.SessionSnapshot{
		ID: session.ID, AccountID: session.AccountID, CreatedAt: session.CreatedAt, ExpiresAt: session.ExpiresAt, RevokedAt: session.RevokedAt,
	})
	if err != nil {
		return err
	}
	if !restoredSession.Authorizes(restoredAccount, now) {
		return service.ErrSessionUnbound
	}
	return nil
}

// trustedClock canonicalizes injected clock samples to UTC milliseconds.
func trustedClock(clock service.Clock) time.Time {
	return clock.Now().UTC().Truncate(time.Millisecond)
}
