package executionport

import (
	"context"
	"errors"
	"time"

	"github.com/rfanazhari/amer-prototype/domain/aggregate/execution"
)

var (
	// ErrNotFound reports that the requested execution or assignment was not stored.
	ErrNotFound = errors.New("execution record not found")
	// ErrAssignmentUnavailable reports that the assignment is revoked or already completed.
	ErrAssignmentUnavailable = errors.New("assignment is unavailable")
	// ErrIdempotencyConflict reports that the same start identity was used with a different intent.
	ErrIdempotencyConflict = errors.New("start identity conflicts with an existing different intent")
	// ErrAlreadyExists reports that a unique key, such as the assignment pin, is already stored.
	ErrAlreadyExists = errors.New("execution record already exists")
	// ErrUncertainCommit reports that the commit result is unknown and the same key may be retried.
	ErrUncertainCommit = errors.New("execution commit result is uncertain")
	// ErrDependencyUnavailable reports that a required external dependency cannot be read.
	ErrDependencyUnavailable = errors.New("a required execution dependency is unavailable")
)

// AssignmentRow carries the assignment facts needed to admit an interview execution.
type AssignmentRow struct {
	// AssignmentID is the assignment identity used as the interview scope.
	AssignmentID string
	// InstanceID is the agent instance identity on the assignment.
	InstanceID string
	// RevisionID is the published revision identity pinned on the assignment.
	RevisionID string
	// AccessRevokedAt is the instant at which assignment access was revoked, nil if still open.
	AccessRevokedAt *time.Time
	// CompletedAt is the interview completion marker, nil if the interview is not complete.
	CompletedAt *time.Time
}

// AssignmentLock acquires an exclusive lock on the assignment row and runs the callback.
// The callback receives the locked assignment facts and a context carrying the open transaction.
// Store calls made with that context join the transaction. The lock and any writes within the callback
// are released when the surrounding transaction ends.
type AssignmentLock interface {
	// WithAssignment locks the assignment by id and runs fn, rolling back on error.
	// ErrNotFound is returned, and fn is not run, when the assignment is absent.
	WithAssignment(ctx context.Context, assignmentID string, fn func(ctx context.Context, row AssignmentRow) error) error
}

// Transactor runs a unit of work in one database transaction without an assignment lock.
type Transactor interface {
	// WithTransaction runs fn in one transaction and rolls back when fn returns an error.
	// Store calls made with the context handed to fn join that transaction.
	WithTransaction(ctx context.Context, fn func(ctx context.Context) error) error
}

// ExecutionStore reads and writes executions within a transaction boundary.
type ExecutionStore interface {
	// Insert writes a new CREATED execution and its initial transition without an interview pin.
	Insert(ctx context.Context, snap execution.Snapshot, fact execution.TransitionFact) error
	// InsertInterview writes the execution and its initial transition, then the assignment pin, atomically.
	// ErrAlreadyExists is returned when the assignment already has a pin, and ErrDependencyUnavailable
	// when the instance or revision is missing.
	InsertInterview(ctx context.Context, assignmentID string, snap execution.Snapshot, fact execution.TransitionFact) error
	// Update locks the execution by id and runs fn; the callback may mutate the snapshot and returns the
	// transition fact to append, or nil when no state change is recorded. The instance and revision binding
	// is never written. A changed snapshot that matches no nonterminal row yields the locked terminal outcome
	// as execution.ErrTerminalConflict. Rolls back on error.
	Update(ctx context.Context, id string, fn func(*execution.Snapshot) (*execution.TransitionFact, error)) error
	// Find returns the stored snapshot by id, or ErrNotFound.
	Find(ctx context.Context, id string) (execution.Snapshot, error)
	// FindByAssignment returns the snapshot linked by the interview pin, or ErrNotFound.
	FindByAssignment(ctx context.Context, assignmentID string) (execution.Snapshot, error)
}

// CommandStore reads and writes idempotent command rows.
type CommandStore interface {
	// FindOrInsert returns the execution id already stored for (scope, kind, key),
	// or inserts a new row pointing at executionID and returns ErrUncertainCommit when ambiguous.
	FindOrInsert(ctx context.Context, scope, kind, key, fingerprint, executionID string) (string, error)
}

// ProgressCommitment answers whether a progress reference is already committed in a sibling store.
type ProgressCommitment interface {
	// IsCommitted returns true when the given progress reference is durably committed
	// for the execution identified by executionID.
	IsCommitted(ctx context.Context, executionID string, ref execution.ProgressRef) (bool, error)
}
