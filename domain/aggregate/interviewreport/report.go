package interviewreport

import "time"

// State is one of the four recognized positions of a report generation.
type State string

const (
	// StatePending is an admitted generation that no attempt has claimed.
	StatePending State = "pending"
	// StateRunning is a generation whose job has claimed an attempt.
	StateRunning State = "running"
	// StateSucceeded is a terminal generation that holds an accepted account.
	StateSucceeded State = "succeeded"
	// StateFailed is a terminal generation that holds a failure reason.
	StateFailed State = "failed"
)

// Generation is a detached copy of one attempt-series to produce or replace the account.
type Generation struct {
	// ID is the stable generation identity.
	ID string
	// Number is the one-based position of the generation within its report.
	Number int
	// State is the current generation position.
	State State
	// FailureReason is set exactly when State is failed.
	FailureReason string
	// CreatedAt is the trusted instant at which the generation was admitted.
	CreatedAt time.Time
	// UpdatedAt is the trusted instant of the most recent change.
	UpdatedAt time.Time
	// Statements is the accepted account, present only on a succeeded generation.
	Statements []Statement
}

// Snapshot is a detached copy of a report with its generations in number order.
type Snapshot struct {
	// ID is the stable report identity.
	ID string
	// ExecutionID is the interview execution the report describes.
	ExecutionID string
	// AssignmentID is the interview assignment the report describes.
	AssignmentID string
	// CreatedAt is the trusted instant at which the report was first committed.
	CreatedAt time.Time
	// Generations holds the full generation history in number order.
	Generations []Generation
}

// Report is the aggregate root of one finished interview's HR account lifecycle.
// It never changes the interview it describes.
type Report struct {
	state Snapshot
}

// NewReport creates a report with no generation.
// It rejects an empty identity or a zero trusted instant.
func NewReport(id, executionID, assignmentID string, now time.Time) (*Report, error) {
	return nil, nil
}

// RestoreReport rebuilds the aggregate from stored rows without calling any provider.
// It rejects inconsistent generation history and statement shape.
func RestoreReport(s Snapshot) (*Report, error) {
	return nil, nil
}

// Snapshot returns a detached copy of the report and its generations.
func (r *Report) Snapshot() Snapshot { return Snapshot{} }

// StartGeneration appends the next pending generation.
// It rejects a start while another generation is pending or running.
func (r *Report) StartGeneration(id string, now time.Time) (Generation, error) {
	return Generation{}, nil
}

// Run moves a pending generation to running when its job claims an attempt.
func (r *Report) Run(generationID string, now time.Time) (Generation, error) {
	return Generation{}, nil
}

// Succeed records the accepted statements and ends a running generation.
func (r *Report) Succeed(generationID string, statements []Statement, now time.Time) (Generation, error) {
	return Generation{}, nil
}

// Fail records a non-empty reason and ends a pending or running generation.
func (r *Report) Fail(generationID, reason string, now time.Time) (Generation, error) {
	return Generation{}, nil
}
