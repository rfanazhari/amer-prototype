package executionport

import (
	"context"
	"errors"
	"time"
)

// ErrAttemptBudget indicates a durably failed execution after attempt capacity expires.
var ErrAttemptBudget = errors.New("execution attempt budget exhausted")

// Attempt carries durable stage-independent authority for an operation.
type Attempt struct {
	ID           string
	ExecutionID  string
	OperationKey string
	Generation   int64
	Number       int
	Outcome      string
	Deadline     time.Time
	CreatedAt    time.Time
}

// AttemptLedger persists attempts under the execution row lock.
type AttemptLedger interface {
	// List returns operation attempts in generation order, or an empty slice.
	List(ctx context.Context, executionID, operationKey string) ([]Attempt, error)
	// Insert writes an attempt and advances its operation authority in the caller's transaction.
	Insert(ctx context.Context, attempt Attempt) error
	// Find returns one execution-owned attempt or ErrNotFound.
	Find(ctx context.Context, executionID, attemptID string) (Attempt, error)
}
