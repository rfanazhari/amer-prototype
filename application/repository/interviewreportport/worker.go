package interviewreportport

import (
	"context"
	"github.com/rfanazhari/amer-prototype/domain/aggregate/interviewreport"
	"time"
)

// JobRef resolves a durable message to its interview evidence identity.
type JobRef struct{ JobID, ExecutionID string }

// AttemptResult carries validated output or a safe failure reason for one fenced attempt.
type AttemptResult struct {
	Claim         ClaimedJob
	Statements    []interviewreport.Statement
	FailureReason string
	FinishedAt    time.Time
}

// WorkerStore persists atomic, report-only worker transitions.
type WorkerStore interface {
	// FindJob resolves evidence identity without holding a transaction over network I/O.
	FindJob(context.Context, string) (JobRef, error)
	// ClaimAttempt atomically waits, expires or claims this job; ErrNothingToClaim also covers duplicates.
	ClaimAttempt(context.Context, string, bool, time.Time) (ClaimedJob, error)
	// FinishAttempt atomically checks the fence, ends the attempt and stores accepted statements only.
	FinishAttempt(context.Context, AttemptResult) error
	// DueJobs lists a bounded set needing evidence recheck, timeout recovery or retry publication.
	DueJobs(context.Context, time.Time, int) ([]JobRef, error)
	// ReconcileJob fences abandoned work and durably republishes eligible work without starting an attempt.
	ReconcileJob(context.Context, string, bool, time.Time) error
}
