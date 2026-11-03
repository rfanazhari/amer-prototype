package interviewreportusecase

import "github.com/rfanazhari/amer-prototype/application/dto"

// AdmitInput identifies the completed interview that is admitted for reporting.
type AdmitInput struct {
	// AssignmentID is the interview assignment whose completion marker was set by the caller.
	AssignmentID string
	// ExecutionID is the interview execution pinned to that assignment.
	ExecutionID string
}

// RegenerateInput is the authorized regeneration command after transport decoding.
type RegenerateInput struct {
	// AssignmentID is the interview assignment whose report is regenerated.
	AssignmentID string
	// IdempotencyKey is the single validated request key.
	IdempotencyKey string
}

// RegenerateResult is the detached regeneration outcome.
type RegenerateResult struct {
	// Generation is the generation created by this request or returned by replay.
	Generation *dto.ReportGeneration
	// Replayed reports that the same key and fingerprint returned the original generation.
	Replayed bool
}
