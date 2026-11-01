package accessport

import (
	"context"
	"errors"
)

// ErrAssignmentAccessNotReady reports that assignment-first collaboration waits on Interview Q2.
var ErrAssignmentAccessNotReady = errors.New("pending")

// AssignmentFacts are the assignment-owned access facts required by invitation and session work.
type AssignmentFacts struct {
	AssignmentID  string
	CandidateID   string
	RevisionID    string
	AccessRevoked bool
}

// AssignmentTx exposes assignment-owned facts after the assignment lock is held.
type AssignmentTx interface {
	// Facts returns current assignment access facts for the locked assignment.
	Facts(ctx context.Context) (AssignmentFacts, error)
}

// AssignmentAccess collaborates with the Interview assignment owner using assignment-first locking.
// Implementations must return ErrAssignmentAccessNotReady until Q2 supplies the actual owner.
type AssignmentAccess interface {
	// WithAssignment locks the assignment first, then runs access collaboration inside that boundary.
	WithAssignment(ctx context.Context, assignmentID string, fn func(AssignmentTx) error) error
}
