package dto

import "time"

// ExecutionView is the detached application read result for a single agent execution.
// Times are UTC millisecond-precision when set; nil means the fact has not yet occurred.
type ExecutionView struct {
	// ID is the stable execution identity minted at admission.
	ID string
	// AssignmentID is the interview assignment identity, empty for non-interview executions.
	AssignmentID string
	// InstanceID is the agent instance identity copied at admission.
	InstanceID string
	// RevisionID is the published revision identity copied at admission.
	RevisionID string
	// State is the current lifecycle position: CREATED, RUNNING, COMPLETED, FAILED, or CANCELLED.
	State string
	// StateVersion is the monotonically increasing version incremented on each transition.
	StateVersion int64
	// ProgressRef is the opaque token for committed progress, empty before any commit.
	ProgressRef string
	// ActivatedAt is the trusted instant of activation, nil before RUNNING.
	ActivatedAt *time.Time
	// TerminalAt is the trusted instant of the terminal transition, nil before termination.
	TerminalAt *time.Time
	// TerminalReason is the reason token for FAILED or CANCELLED, empty otherwise.
	TerminalReason string
	// CreatedAt is the trusted instant at which the execution was first committed.
	CreatedAt time.Time
	// UpdatedAt is the trusted instant of the most recent mutation.
	UpdatedAt time.Time
}
