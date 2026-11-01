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
	if id == "" || assignmentID == "" || createdBy == "" {
		return nil, ErrInvalidInvitation
	}
	lifetime, err := valueobject.NewSessionLifetime(now, valueobject.InvitationDuration)
	if err != nil {
		return nil, ErrInvalidInvitation
	}
	return &Invitation{id: id, assignmentID: assignmentID, createdBy: createdBy, lifetime: lifetime}, nil
}

// RestoreInvitation validates stored identity, assignment reference and time order without resetting the deadline.
func RestoreInvitation(s InvitationSnapshot) (*Invitation, error) {
	if s.ID == "" || s.AssignmentID == "" || s.CreatedBy == "" {
		return nil, ErrInvalidInvitation
	}
	lifetime, err := valueobject.RestoreSessionLifetime(s.CreatedAt, s.ExpiresAt)
	if err != nil {
		return nil, ErrInvalidInvitation
	}
	if s.RevokedAt != nil && (s.RevokedAt.IsZero() || s.RevokedAt.Before(s.CreatedAt)) {
		return nil, ErrInvalidInvitation
	}
	return &Invitation{
		id:           s.ID,
		assignmentID: s.AssignmentID,
		createdBy:    s.CreatedBy,
		lifetime:     lifetime,
		revokedAt:    copyTime(s.RevokedAt),
	}, nil
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
	return i != nil && i.id != "" && i.revokedAt == nil && i.lifetime.Current(now)
}

// Revoke records trusted revocation time on a valid unrevoked invitation.
func (i *Invitation) Revoke(now time.Time) error {
	if i == nil || i.id == "" || i.assignmentID == "" || !i.lifetime.Valid() || i.revokedAt != nil || now.IsZero() || now.Before(i.lifetime.CreatedAt()) {
		return ErrInvalidInvitation
	}
	i.revokedAt = copyTime(&now)
	return nil
}

// AssignmentID returns the assignment this invitation is scoped to.
func (i *Invitation) AssignmentID() string {
	if i == nil {
		return ""
	}
	return i.assignmentID
}
