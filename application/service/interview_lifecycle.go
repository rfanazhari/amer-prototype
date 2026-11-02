package service

import (
	"context"
	"time"
)

// InterviewAttempt identifies the lifecycle authority required for an interview write.
// ExpectedVersion is the execution version, separate from the progress store version.
// No stage, question or other HR policy crosses this boundary.
type InterviewAttempt struct {
	ExecutionID     string
	AttemptID       string
	ExpectedVersion int64
}

// InterviewActivation requests an initial or recovery attempt on the same execution.
// Intent is activate or reconnect. Detection time is backend-owned for reconnect.
// IdempotencyKey identifies a replay of the same intent; budgets are passed explicitly.
type InterviewActivation struct {
	Authority      InterviewAttempt
	Intent         string
	IdempotencyKey string
	DetectedAt     *time.Time
	Budgets        InterviewBudgets
}

// InterviewActivationResult returns the admitted attempt and trusted lifecycle timing.
// Replayed is true for a previously recorded identical intent.
type InterviewActivationResult struct {
	Authority    InterviewAttempt
	RunningAt    time.Time
	HardDeadline time.Time
	Replayed     bool
}

// InterviewCompletion references a progress record already durably committed by the interview.
// ProgressReference is opaque to the lifecycle; completion never creates a report.
type InterviewCompletion struct {
	Authority         InterviewAttempt
	ProgressReference string
}

// InterviewFailure requests failure without removing already committed interview records.
type InterviewFailure struct {
	Authority InterviewAttempt
	Reason    string
}

// InterviewLifecycle is the interview-owned contract for the stage-agnostic execution authority.
// This declaration provides no real fence; a lifecycle adapter must supply its guarantees.
type InterviewLifecycle interface {
	// Activate records or replays an attempt under the supplied activation or reconnect limits.
	// Exhaustion fails the same execution; it never opens another execution.
	Activate(ctx context.Context, req InterviewActivation) (InterviewActivationResult, error)
	// Fence checks the expected version and current attempt without authorizing a later standalone write.
	Fence(ctx context.Context, authority InterviewAttempt) error
	// WithProgress locks the execution before child rows, checks nonterminal state, version and
	// current attempt, then runs write in that same transaction. Stores join via the callback context.
	// Rejection never invokes write. Callback failure rolls back all writes and releases the lock.
	WithProgress(ctx context.Context, authority InterviewAttempt, write func(context.Context) error) error
	// Complete checks the committed opaque progress reference and applies the first terminal commit.
	Complete(ctx context.Context, req InterviewCompletion) error
	// Fail applies the first terminal commit while preserving previously committed progress.
	Fail(ctx context.Context, req InterviewFailure) error
}
