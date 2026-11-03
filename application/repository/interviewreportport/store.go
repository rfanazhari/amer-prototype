package interviewreportport

import (
	"context"
	"errors"
	"time"

	"github.com/rfanazhari/amer-prototype/domain/aggregate/interviewreport"
)

var (
	// ErrNotFound reports that the requested assignment, execution or report was not stored.
	ErrNotFound = errors.New("interview report record not found")
	// ErrDependencyUnavailable reports that a required store cannot be read or written.
	ErrDependencyUnavailable = errors.New("interview report store is unavailable")
)

// Transaction runs a unit of work in one database transaction without exposing the database handle.
type Transaction interface {
	// InTransaction runs fn in one transaction and rolls back when fn returns an error.
	// When ctx already carries an open transaction, fn joins it and the caller keeps ownership of the commit.
	// Port calls made with the context handed to fn join that transaction.
	InTransaction(ctx context.Context, fn func(ctx context.Context) error) error
}

// CompletionMarker reads the sibling interview completion marker inside the caller's transaction.
type CompletionMarker interface {
	// CompletedAt returns the instant at which the assignment was completed, or nil while it is not.
	// ErrNotFound is returned when the assignment is unknown or the execution does not belong to it.
	CompletedAt(ctx context.Context, assignmentID, executionID string) (*time.Time, error)
}

// ReportReader loads the stored report lifecycle.
type ReportReader interface {
	// FindByExecution returns the report with its generations in number order, or ErrNotFound.
	FindByExecution(ctx context.Context, executionID string) (interviewreport.Snapshot, error)
}

// Admission is the durable intent recorded when a completed interview is admitted for reporting.
type Admission struct {
	// Report is the new report holding generation 1 in the pending state.
	Report interviewreport.Snapshot
	// JobID identifies the background job of the first generation.
	JobID string
	// OutboxID identifies the outbox message that will publish the job.
	OutboxID string
	// WaitDeadlineAt is the instant at which a job still waiting for final evidence gives up.
	WaitDeadlineAt time.Time
}

// AdmissionWriter stores an admission in the caller's transaction.
type AdmissionWriter interface {
	// Insert writes the report, its first generation, the job and one outbox row.
	// It does not update the assignment or the execution and does not publish to a broker.
	Insert(ctx context.Context, admission Admission) error
}
