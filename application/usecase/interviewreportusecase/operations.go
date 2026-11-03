package interviewreportusecase

import (
	"context"

	"github.com/rfanazhari/amer-prototype/application/dto"
)

// AdmitOperation is the in-process completion port. It runs inside the caller's open transaction
// and is not an HTTP route.
type AdmitOperation interface {
	// Execute inserts the first pending generation, job and outbox row, or returns the existing first generation.
	// It never commits, publishes to a broker or updates the assignment or execution.
	Execute(context.Context, AdmitInput) (*dto.ReportGeneration, error)
}

// RegenerateOperation is the explicit idempotent regeneration command consumed by delivery.
type RegenerateOperation interface {
	// Execute authorizes current HR, checks replay first, and returns a detached generation only after commit.
	Execute(context.Context, RegenerateInput) (*RegenerateResult, error)
}
