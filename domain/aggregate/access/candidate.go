package access

import "time"

// CandidateSnapshot transfers persisted candidate identity without exposing the live aggregate.
type CandidateSnapshot struct {
	ID        string
	Name      string
	Email     string
	CreatedBy string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// Candidate owns reusable person identity without administrative credentials or roles.
type Candidate struct{ state CandidateSnapshot }

// NewCandidate records a new identity with caller-supplied contact already accepted by application policy.
func NewCandidate(id, name, email, createdBy string, now time.Time) (*Candidate, error) {
	return nil, ErrInvalidCandidate
}

// RestoreCandidate validates stored identity, creator and timestamp order without rewriting contact history.
func RestoreCandidate(s CandidateSnapshot) (*Candidate, error) {
	return nil, ErrInvalidCandidate
}

// Snapshot returns a detached copy of the current candidate facts.
func (c *Candidate) Snapshot() CandidateSnapshot {
	if c == nil {
		return CandidateSnapshot{}
	}
	return c.state
}

// SubjectID returns the stable candidate identity used across assignments.
func (c *Candidate) SubjectID() string {
	if c == nil {
		return ""
	}
	return c.state.ID
}
