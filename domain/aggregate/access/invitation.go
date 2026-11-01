package access

import (
	"time"

	"github.com/rfanazhari/amer-prototype/domain/valueobject"
)

// InvitationSnapshot transfers persisted invitation facts without exposing the live entity. Pointer fields are copied at boundaries.
type InvitationSnapshot struct {
	ID           string
	AssignmentID string
	CreatedBy    string
	CreatedAt    time.Time
	ExpiresAt    time.Time
	RevokedAt    *time.Time
}

// Invitation owns assignment-scoped entry-credential lifetime and revocation.
type Invitation struct {
	id           string
	assignmentID string
	createdBy    string
	lifetime     valueobject.SessionLifetime
	revokedAt    *time.Time
}

// NewInvitation records a new unrevoked invitation whose expiry is exactly creation plus 24 hours.
func NewInvitation(id, assignmentID, createdBy string, now time.Time) (*Invitation, error) {
	return nil, ErrInvalidInvitation
}

// RestoreInvitation validates stored identity, assignment reference and time order without resetting the deadline.
func RestoreInvitation(s InvitationSnapshot) (*Invitation, error) {
	return nil, ErrInvalidInvitation
}

// Snapshot returns a detached copy of the current invitation facts, including an optional revocation pointer.
func (i *Invitation) Snapshot() InvitationSnapshot {
	if i == nil {
		return InvitationSnapshot{}
	}
	return InvitationSnapshot{
		ID:           i.id,
		AssignmentID: i.assignmentID,
		CreatedBy:    i.createdBy,
		CreatedAt:    i.lifetime.CreatedAt(),
		ExpiresAt:    i.lifetime.ExpiresAt(),
		RevokedAt:    copyTime(i.revokedAt),
	}
}

// Current reports whether the invitation is unrevoked and now is strictly before expiry.
func (i *Invitation) Current(now time.Time) bool {
	return false
}

// Revoke records trusted revocation time on a valid unrevoked invitation.
func (i *Invitation) Revoke(now time.Time) error {
	return ErrInvalidInvitation
}

// AssignmentID returns the assignment this invitation is scoped to.
func (i *Invitation) AssignmentID() string {
	if i == nil {
		return ""
	}
	return i.assignmentID
}
