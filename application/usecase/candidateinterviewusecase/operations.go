package candidateinterviewusecase

import (
	"context"

	"github.com/rfanazhari/amer-prototype/application/dto"
)

// AcceptNoticeInput carries the acceptance request of the resolved candidate session.
type AcceptNoticeInput struct {
	// NoticeVersion is the version the candidate saw.
	NoticeVersion string
	// Accepted must be true; false is malformed input, not withdrawal.
	Accepted bool
}

// AcceptNoticeResult is the stored acceptance.
type AcceptNoticeResult struct {
	// Consent is the acceptance, new or found.
	Consent dto.ConsentView
	// Created is true on first acceptance and false on a retry.
	Created bool
}

// AcceptNoticeOperation records acceptance of the current notice version.
type AcceptNoticeOperation interface {
	// Execute stores or finds the acceptance; an empty catalog writes nothing and reports a dependency failure.
	Execute(ctx context.Context, in AcceptNoticeInput) (*AcceptNoticeResult, error)
}

// StartInterviewInput carries one candidate start.
type StartInterviewInput struct {
	// ConsentID is the stored acceptance the start relies on.
	ConsentID string
	// IdempotencyKey makes a lost start repeatable.
	IdempotencyKey string
	// ControllerID is the calling tab, taken from X-Interview-Controller.
	ControllerID string
}

// StartInterviewResult is the execution a start committed or found.
type StartInterviewResult struct {
	// Execution is the candidate-safe execution.
	Execution dto.InterviewExecutionView
	// Created is true when this call committed the execution.
	Created bool
}

// StartInterviewOperation checks consent and coverage, then asks lifecycle to commit or find the execution.
type StartInterviewOperation interface {
	// Execute returns before the lifecycle port when consent or coverage fails.
	Execute(ctx context.Context, in StartInterviewInput) (*StartInterviewResult, error)
}

// TakeoverInput carries one controller takeover.
type TakeoverInput struct {
	// ExecutionID is the execution being controlled.
	ExecutionID string
	// ControllerID is the new controlling tab; it equals X-Interview-Controller.
	ControllerID string
	// ExpectedVersion is the controller version the caller last saw.
	ExpectedVersion int64
}

// TakeoverOperation replaces the controlling tab.
type TakeoverOperation interface {
	// Execute replaces the controller, or reports a conflict for a stale version and leaves the previous id.
	Execute(ctx context.Context, in TakeoverInput) (*dto.TakeoverView, error)
}

// CancelInterviewInput carries one early stop.
type CancelInterviewInput struct {
	// ExecutionID is the execution to cancel.
	ExecutionID string
	// ControllerID is the calling tab, which must be the current controller.
	ControllerID string
	// ExpectedVersion is the execution state version the caller last saw.
	ExpectedVersion int64
}

// CancelInterviewOperation asks lifecycle to commit a candidate early stop.
type CancelInterviewOperation interface {
	// Execute returns the execution as CANCELLED; a non-controller never reaches the lifecycle port.
	Execute(ctx context.Context, in CancelInterviewInput) (*dto.InterviewExecutionView, error)
}
