package candidateinterviewquery

import (
	"context"

	"github.com/rfanazhari/amer-prototype/application/dto"
)

// GetInterviewInput identifies the calling tab for the screen read.
type GetInterviewInput struct {
	// ControllerID is the calling tab from X-Interview-Controller, empty when absent.
	ControllerID string
}

// GetInterviewOperation assembles the screen model for the resolved candidate session.
type GetInterviewOperation interface {
	// Execute reads notice, consent, execution and progress; it never starts an interview or inserts consent.
	Execute(ctx context.Context, in GetInterviewInput) (*dto.InterviewView, error)
}

// GetExecutionInput identifies one execution and the calling tab.
type GetExecutionInput struct {
	// ExecutionID is the execution requested by the client.
	ExecutionID string
	// ControllerID is the calling tab from X-Interview-Controller, empty when absent.
	ControllerID string
}

// GetExecutionOperation reads one execution inside the session's assignment.
type GetExecutionOperation interface {
	// Execute returns the execution, or a not-found error when it is outside the assignment.
	Execute(ctx context.Context, in GetExecutionInput) (*dto.InterviewExecutionView, error)
}
