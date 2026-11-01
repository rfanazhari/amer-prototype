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
	if id == "" || name == "" || email == "" || createdBy == "" || now.IsZero() {
		return nil, ErrInvalidCandidate
	}
	return &Candidate{state: CandidateSnapshot{
		ID:        id,
		Name:      name,
		Email:     email,
		CreatedBy: createdBy,
		CreatedAt: now,
		UpdatedAt: now,
	}}, nil
}

// RestoreCandidate validates stored identity, creator and timestamp order without rewriting contact history.
func RestoreCandidate(s CandidateSnapshot) (*Candidate, error) {
	if s.ID == "" || s.Name == "" || s.Email == "" || s.CreatedBy == "" || s.CreatedAt.IsZero() || s.UpdatedAt.Before(s.CreatedAt) {
		return nil, ErrInvalidCandidate
	}
	return &Candidate{state: s}, nil
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
