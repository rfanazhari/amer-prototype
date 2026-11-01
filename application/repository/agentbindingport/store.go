package agentbindingport

import (
	"context"
	"errors"
)

var (
	// ErrNotFound reports that an assignment or execution binding was not stored.
	ErrNotFound = errors.New("configuration binding record not found")
	// ErrNoActiveRevision reports that a new binding cannot select an unpublished instance.
	ErrNoActiveRevision = errors.New("instance has no active published revision")
	// ErrUnsupportedTemplate reports that a new assignment cannot pin a non-HR Interviewer instance.
	ErrUnsupportedTemplate = errors.New("assignment requires an hr-interviewer instance")
)

// AssignmentBinding is an Interview-owned assignment pin to one immutable published revision.
type AssignmentBinding struct {
	AssignmentID string
	CandidateID  string
	InstanceID   string
	RevisionID   string
}

// ExecutionBinding is an Interview-owned execution pin copied from its assignment at start.
type ExecutionBinding struct {
	ExecutionID  string
	AssignmentID string
	InstanceID   string
	RevisionID   string
}

// Assignments persists assignment revision pins without mutating published configuration.
type Assignments interface {
	// Insert stores a new assignment pin to an already-published revision.
	Insert(context.Context, AssignmentBinding) error
	// Find restores one assignment pin or returns ErrNotFound.
	Find(context.Context, string) (AssignmentBinding, error)
}

// Executions persists execution revision pins copied at start time.
type Executions interface {
	// Insert stores a new execution pin that must not follow later active selection.
	Insert(context.Context, ExecutionBinding) error
	// Find restores one execution pin or returns ErrNotFound.
	Find(context.Context, string) (ExecutionBinding, error)
}
