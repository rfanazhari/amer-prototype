package interviewreportport

import (
	"context"
	"errors"
	"time"

	"github.com/rfanazhari/amer-prototype/domain/aggregate/interviewreport"
)

// ErrDuplicateCommand reports a unique command-identity loser that must reconcile outside the aborted transaction.
var ErrDuplicateCommand = errors.New("interview report command identity already exists")

// AssignmentRef is the interview assignment with the execution pinned to it and the sibling completion marker.
type AssignmentRef struct {
	// AssignmentID is the interview assignment.
	AssignmentID string
	// ExecutionID is the single execution pinned to the assignment.
	ExecutionID string
	// CompletedAt is the sibling completion instant, nil while the interview is not completed.
	CompletedAt *time.Time
}

// AssignmentReader resolves an assignment without locking it.
type AssignmentReader interface {
	// FindAssignment returns the assignment and its execution pin, or ErrNotFound when either is unknown.
	FindAssignment(ctx context.Context, assignmentID string) (AssignmentRef, error)
}

// AssignmentLocker locks an assignment for a path that also writes report rows.
type AssignmentLocker interface {
	// LockAssignment locks the assignment row and then its execution row, in that order, inside the
	// caller's transaction. It returns ErrNotFound when either is unknown.
	LockAssignment(ctx context.Context, assignmentID string) (AssignmentRef, error)
}

// CommandIdentity is the already-authorized identity that makes a regeneration request idempotent.
type CommandIdentity struct {
	// Scope is the administrative subject that issued the command.
	Scope string
	// Kind is the durable command kind.
	Kind string
	// Key is the validated request key.
	Key string
}

// StoredCommand is the original outcome retained for replay.
type StoredCommand struct {
	// Fingerprint is the lowercase hex SHA-256 of the original intent.
	Fingerprint string
	// GenerationID is the generation the command created.
	GenerationID string
	// ExecutionID is the execution whose report holds that generation.
	ExecutionID string
}

// Regeneration is the durable intent recorded when HR asks for a new generation.
type Regeneration struct {
	// Command is the idempotency identity.
	Command CommandIdentity
	// CommandID is the identity of the command row.
	CommandID string
	// Fingerprint is the lowercase hex SHA-256 of the intent.
	Fingerprint string
	// CreatesReport reports that the report row does not exist yet and is inserted with the generation.
	CreatesReport bool
	// Report holds the report with the new pending generation as its last generation.
	Report interviewreport.Snapshot
	// JobID identifies the background job of the new generation.
	JobID string
	// OutboxID identifies the outbox message that will publish the job.
	OutboxID string
	// WaitDeadlineAt is the instant at which a job still waiting for final evidence gives up.
	WaitDeadlineAt time.Time
}

// RegenerationStore stores regeneration commands and their generations.
type RegenerationStore interface {
	// FindCommand returns the original command and whether it exists.
	FindCommand(ctx context.Context, identity CommandIdentity) (StoredCommand, bool, error)
	// InsertRegeneration writes the command, the report when it is new, the generation, the job and one
	// outbox row in the caller's transaction. It returns ErrDuplicateCommand when the command identity
	// is already stored and interviewreport.ErrGenerationActive when another generation became active.
	// It never updates the assignment or the execution and never publishes to a broker.
	InsertRegeneration(ctx context.Context, regeneration Regeneration) error
}
