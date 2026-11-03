package interviewreportport

import (
	"errors"
	"time"
)

// ErrNothingToClaim reports that no job or outbox row was ready to claim.
var ErrNothingToClaim = errors.New("no interview report work is ready to claim")

// ClaimedJob is the attempt a worker holds after a job claim commits.
type ClaimedJob struct {
	// JobID identifies the claimed job.
	JobID string
	// GenerationID identifies the generation the job belongs to.
	GenerationID string
	// AttemptNumber is the one-based number of the attempt inserted by the claim.
	AttemptNumber int
	// FencingToken is the job token the attempt carries, so an older holder can be refused at commit.
	FencingToken int64
	// StartedAt is the trusted claim instant.
	StartedAt time.Time
	// AttemptDeadlineAt is the earlier of the attempt cap and the job deadline.
	AttemptDeadlineAt time.Time
	// JobDeadlineAt is the end of the job window that began at the first claim.
	JobDeadlineAt time.Time
}

// OutboxMessage is one unconfirmed publish intent claimed for publication.
type OutboxMessage struct {
	// ID is the message identity.
	ID string
	// JobKind names the kind of job the message points at.
	JobKind string
	// JobID identifies the job to run.
	JobID string
	// CreatedAt is the instant the intent was recorded.
	CreatedAt time.Time
}
