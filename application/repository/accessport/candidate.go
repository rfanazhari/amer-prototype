package accessport

import (
	"context"
	"time"
)

// CandidateRecord is the detached candidate identity persisted independently of the domain aggregate.
type CandidateRecord struct {
	ID        string
	Name      string
	Email     string
	CreatedBy string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// InvitationRecord is the detached assignment invitation persisted independently of the domain entity.
type InvitationRecord struct {
	ID           string
	AssignmentID string
	TokenHash    string
	CreatedBy    string
	CreatedAt    time.Time
	ExpiresAt    time.Time
	RevokedAt    *time.Time
}

// CandidateSessionRecord is the detached candidate session persisted independently of the domain entity.
type CandidateSessionRecord struct {
	ID           string
	AssignmentID string
	InvitationID string
	TokenHash    string
	CreatedAt    time.Time
	ExpiresAt    time.Time
	RevokedAt    *time.Time
}

// Candidates persists reusable candidate identity without uniqueness on contact email.
type Candidates interface {
	// Insert stores a new candidate identity.
	Insert(ctx context.Context, record CandidateRecord) error
	// FindByID restores one candidate by stable identity or returns ErrNotFound.
	FindByID(ctx context.Context, id string) (CandidateRecord, error)
}

// Invitations reads invitation credentials outside the assignment lock for exchange lookup.
type Invitations interface {
	// FindByTokenHash restores one invitation by digest or returns ErrNotFound.
	FindByTokenHash(ctx context.Context, tokenHash string) (InvitationRecord, error)
}

// CandidateSessions reads candidate session lifetime by credential digest.
type CandidateSessions interface {
	// FindByTokenHash restores one candidate session by digest or returns ErrNotFound.
	FindByTokenHash(ctx context.Context, tokenHash string) (CandidateSessionRecord, error)
}
