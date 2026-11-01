package candidateinterviewport

import (
	"context"
	"errors"

	"github.com/rfanazhari/amer-prototype/domain/aggregate/candidateinterview"
)

var (
	// ErrNotFound reports that the consent or controller was not stored.
	ErrNotFound = errors.New("candidate interview record not found")
	// ErrAlreadyExists reports a unique-key violation, such as a second acceptance of one notice version.
	ErrAlreadyExists = errors.New("candidate interview record already exists")
	// ErrAssignmentUnavailable reports an absent or revoked assignment, or an ended candidate session.
	ErrAssignmentUnavailable = errors.New("assignment or candidate session is unavailable")
)

// ConsentTransaction runs consent writes inside the assignment boundary.
type ConsentTransaction interface {
	// WithAssignment locks the assignment, re-reads revocation and the candidate session, then runs fn.
	// Store calls made with the context handed to fn join the transaction. It returns ErrAssignmentUnavailable
	// without running fn when the assignment is revoked or the session has ended, and rolls back when fn fails.
	WithAssignment(ctx context.Context, assignmentID, sessionID string, fn func(ctx context.Context) error) error
}

// Consents reads and writes insert-only acceptances.
type Consents interface {
	// Insert stores a new acceptance, or returns ErrAlreadyExists for the same assignment and notice version.
	Insert(ctx context.Context, snap candidateinterview.ConsentSnapshot) error
	// Find returns the acceptance of one notice version for the assignment, or ErrNotFound.
	Find(ctx context.Context, assignmentID string, version candidateinterview.NoticeVersion) (candidateinterview.ConsentSnapshot, error)
	// FindByID returns the acceptance with the id inside the assignment, or ErrNotFound.
	FindByID(ctx context.Context, assignmentID, id string) (candidateinterview.ConsentSnapshot, error)
}

// Controllers reads and writes the single controlling tab of an assignment.
type Controllers interface {
	// Find returns the controller of the assignment, or ErrNotFound.
	Find(ctx context.Context, assignmentID string) (candidateinterview.ControllerSnapshot, error)
	// Claim stores the first controller, or returns ErrAlreadyExists when the assignment has one.
	Claim(ctx context.Context, snap candidateinterview.ControllerSnapshot) error
	// Update locks the assignment's controller row and runs fn on the restored aggregate; when fn succeeds the
	// store writes controller id, version and updated_at explicitly. ErrNotFound is returned without running fn.
	Update(ctx context.Context, assignmentID string, fn func(*candidateinterview.InterviewController) error) error
}
