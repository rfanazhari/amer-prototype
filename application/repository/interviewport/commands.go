package interviewport

import (
	"context"
	"time"

	"github.com/rfanazhari/amer-prototype/domain/aggregate/agentconfig"
	"github.com/rfanazhari/amer-prototype/domain/valueobject"
)

// SessionCommand is the stored identity of one accepted session-attempt request.
// Key is already scoped to the candidate session. AnswerSDP is the provider answer returned on the first request.
type SessionCommand struct {
	// State is pending, succeeded, failed or legacy.
	State            string
	ID               string
	ExecutionID      string
	Key              string
	Intent           string
	AttemptID        string
	SessionID        string
	AnswerSDP        string
	ExecutionVersion int64
	CreatedAt        time.Time
}

// SessionCommands persists accepted session-attempt requests so a repeated key replays the original result.
type SessionCommands interface {
	// Find returns the command stored for the execution and scoped key, or ErrNotFound.
	Find(ctx context.Context, executionID, key string) (SessionCommand, error)
	// Insert stores a command; a duplicate execution and key yields ErrConflict.
	Insert(ctx context.Context, command SessionCommand) error
	// Finalize stores the original answer once; pending identity and session ownership must match.
	Finalize(ctx context.Context, command SessionCommand) error
	// Fail marks a known setup failure without ever permitting provider recreation for that key.
	Fail(ctx context.Context, executionID, key string) error
}

// RevisionReader resolves the immutable revision an execution is pinned to.
type RevisionReader interface {
	// Find returns the pinned published HR revision for the execution; absence yields ErrNotFound.
	Find(ctx context.Context, executionID string) (agentconfig.PublishedRevision[valueobject.HRConfiguration], error)
}
