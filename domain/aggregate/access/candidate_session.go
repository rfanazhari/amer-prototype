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
	return nil, ErrInvalidCandidateSession
}

// RestoreCandidateSession validates stored identity, assignment and invitation references without resetting the deadline.
func RestoreCandidateSession(s CandidateSessionSnapshot) (*CandidateSession, error) {
	return nil, ErrInvalidCandidateSession
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
	return false
}

// Authorizes reports whether this current session is bound to assignmentID.
func (s *CandidateSession) Authorizes(assignmentID string, now time.Time) bool {
	return false
}
