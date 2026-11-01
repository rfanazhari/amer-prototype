package access

import (
	"time"

	"github.com/rfanazhari/amer-prototype/domain/valueobject"
)

// CandidateSessionSnapshot transfers persisted candidate session facts without exposing the live entity.
type CandidateSessionSnapshot struct {
	ID           string
	AssignmentID string
	InvitationID string
	CreatedAt    time.Time
	ExpiresAt    time.Time
	RevokedAt    *time.Time
}

// CandidateSession owns assignment-scoped candidate access lifetime and revocation.
type CandidateSession struct {
	id           string
	assignmentID string
	invitationID string
	lifetime     valueobject.SessionLifetime
	revokedAt    *time.Time
}

// NewCandidateSession records a new unrevoked session whose expiry is exactly creation plus two hours.
func NewCandidateSession(id, assignmentID, invitationID string, now time.Time) (*CandidateSession, error) {
	if id == "" || assignmentID == "" || invitationID == "" {
		return nil, ErrInvalidCandidateSession
	}
	lifetime, err := valueobject.NewSessionLifetime(now, valueobject.AdministrativeDuration)
	if err != nil {
		return nil, ErrInvalidCandidateSession
	}
	return &CandidateSession{id: id, assignmentID: assignmentID, invitationID: invitationID, lifetime: lifetime}, nil
}

// RestoreCandidateSession validates stored identity, assignment and invitation references without resetting the deadline.
func RestoreCandidateSession(s CandidateSessionSnapshot) (*CandidateSession, error) {
	if s.ID == "" || s.AssignmentID == "" || s.InvitationID == "" {
		return nil, ErrInvalidCandidateSession
	}
	lifetime, err := valueobject.RestoreSessionLifetime(s.CreatedAt, s.ExpiresAt)
	if err != nil {
		return nil, ErrInvalidCandidateSession
	}
	if s.RevokedAt != nil && (s.RevokedAt.IsZero() || s.RevokedAt.Before(s.CreatedAt)) {
		return nil, ErrInvalidCandidateSession
	}
	return &CandidateSession{
		id:           s.ID,
		assignmentID: s.AssignmentID,
		invitationID: s.InvitationID,
		lifetime:     lifetime,
		revokedAt:    copyTime(s.RevokedAt),
	}, nil
}

// Snapshot returns a detached copy of the current session facts, including an optional revocation pointer.
func (s *CandidateSession) Snapshot() CandidateSessionSnapshot {
	if s == nil {
		return CandidateSessionSnapshot{}
	}
	return CandidateSessionSnapshot{
		ID:           s.id,
		AssignmentID: s.assignmentID,
		InvitationID: s.invitationID,
		CreatedAt:    s.lifetime.CreatedAt(),
		ExpiresAt:    s.lifetime.ExpiresAt(),
		RevokedAt:    copyTime(s.revokedAt),
	}
}

// Current reports whether the session is unrevoked and now is strictly before expiry.
func (s *CandidateSession) Current(now time.Time) bool {
	return s != nil && s.id != "" && s.revokedAt == nil && s.lifetime.Current(now)
}

// Authorizes reports whether this current session is bound to assignmentID.
func (s *CandidateSession) Authorizes(assignmentID string, now time.Time) bool {
	return s.Current(now) && assignmentID != "" && s.assignmentID == assignmentID
}
