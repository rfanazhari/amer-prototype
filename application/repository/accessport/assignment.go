package accessport

import (
	"context"
	"errors"
)

// ErrAssignmentAccessNotReady reports that a runtime Interview adapter is not wired.
var ErrAssignmentAccessNotReady = errors.New("assignment access collaboration is not ready until interview Q2")

// AssignmentFacts are the Interview-owned access facts required by invitation and session work.
type AssignmentFacts struct {
	AssignmentID      string
	CandidateID       string
	RevisionID        string
	AccessRevoked     bool
	TerminalCompleted bool
}

// liveAssignment reports a recognized, unrevoked assignment. Terminal completion does not deny access.
func (f AssignmentFacts) liveAssignment() bool {
	return f.AssignmentID != "" && f.CandidateID != "" && f.RevisionID != "" && !f.AccessRevoked
}

// PermitsInvitation reports whether HR may issue or reissue an invitation for candidateID on this assignment.
func (f AssignmentFacts) PermitsInvitation(candidateID string) bool {
	return f.liveAssignment() && f.BoundToCandidate(candidateID)
}

// PermitsExchange reports whether a live invitation may create a candidate session for this assignment.
func (f AssignmentFacts) PermitsExchange() bool {
	return f.liveAssignment()
}

// PermitsSessionAccess reports whether an existing candidate session may continue for this assignment.
func (f AssignmentFacts) PermitsSessionAccess() bool {
	return f.liveAssignment()
}

// BoundToCandidate reports whether the locked assignment references candidateID.
func (f AssignmentFacts) BoundToCandidate(candidateID string) bool {
	return candidateID != "" && f.CandidateID == candidateID
}

// AssignmentTx exposes assignment-owned facts after the assignment lock is held.
type AssignmentTx interface {
	// Facts returns current assignment access facts for the locked assignment.
	Facts(ctx context.Context) (AssignmentFacts, error)
}

// AssignmentAccess collaborates with the Interview assignment owner using assignment-first locking.
// Access never creates an assignment table; Interview supplies the adapter after T14.
type AssignmentAccess interface {
	// WithAssignment locks the assignment first, then runs access collaboration inside that boundary.
	WithAssignment(ctx context.Context, assignmentID string, fn func(AssignmentTx) error) error
}
