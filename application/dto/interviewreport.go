package dto

import "time"

// ReportGeneration is a detached view of one report generation.
type ReportGeneration struct {
	// ID is the stable generation identity.
	ID string
	// Number is the one-based position of the generation within its report.
	Number int
	// State is pending, running, succeeded or failed.
	State string
	// FailureReason is set exactly when State is failed.
	FailureReason *string
	// CreatedAt is the UTC instant at which the generation was admitted.
	CreatedAt time.Time
}
