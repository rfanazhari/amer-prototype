package service

import (
	"context"
	"time"

	"github.com/rfanazhari/amer-prototype/domain/aggregate/candidateinterview"
)

// CancelReasonCandidateEarlyStop is the fixed lifecycle reason for a candidate-initiated cancellation.
const CancelReasonCandidateEarlyStop = "candidate_early_stop"

// Notice is the statement and version a candidate is asked to accept.
type Notice struct {
	// Version is the notice version the acceptance is recorded against.
	Version candidateinterview.NoticeVersion
	// Statement is the approved notice text shown to the candidate.
	Statement string
}

// NoticeCatalog supplies the current notice. It is a composition value, not a stored row.
type NoticeCatalog interface {
	// Current returns the current notice, or false when the catalog is empty and acceptance must fail closed.
	Current(ctx context.Context) (Notice, bool)
}

// ExecutionFacts is the execution state Execution Lifecycle reports to the candidate interview.
type ExecutionFacts struct {
	// ID is the stable execution identity.
	ID string
	// AssignmentID is the assignment the execution belongs to.
	AssignmentID string
	// State is CREATED, RUNNING, COMPLETED, FAILED or CANCELLED.
	State string
	// Version is the execution state version used as the expected version for cancel.
	Version int64
	// HardDeadline is the instant the interview must end, nil until the execution is RUNNING.
	HardDeadline *time.Time
}

// CommitOrFindRequest carries the intent of one candidate start.
type CommitOrFindRequest struct {
	// AssignmentID is the assignment taken from the candidate session.
	AssignmentID string
	// IdempotencyKey is the caller-supplied key that makes a lost start repeatable.
	IdempotencyKey string
	// ConsentID is the stored acceptance the start relies on; with the assignment it forms the intent fingerprint.
	ConsentID string
}

// CommitOrFindResult is the execution a start committed or found.
type CommitOrFindResult struct {
	// Execution is the one execution for the assignment.
	Execution ExecutionFacts
	// Created is true when this call committed the execution and false when it was found.
	Created bool
}

// CancelRequest carries one cancellation commit.
type CancelRequest struct {
	// ExecutionID is the execution to cancel.
	ExecutionID string
	// ExpectedVersion is the execution state version the caller last saw.
	ExpectedVersion int64
	// Reason is the lifecycle reason token, CancelReasonCandidateEarlyStop for a candidate stop.
	Reason string
}

// ExecutionLifecycle is the sibling Execution Lifecycle contract the candidate interview calls.
// Failures use the classified InterviewError sentinels; an unbound adapter returns ErrInterviewDependencyUnavailable.
type ExecutionLifecycle interface {
	// CommitOrFind commits the one execution for the assignment, or finds it for the same idempotency intent.
	// The same key with a different consent id reports ErrInterviewConflict.
	CommitOrFind(ctx context.Context, req CommitOrFindRequest) (CommitOrFindResult, error)
	// Find reads the execution for the assignment without creating one; absence is ErrInterviewNotFound.
	Find(ctx context.Context, assignmentID string) (ExecutionFacts, error)
	// Cancel commits CANCELLED; a stale version or an already terminal execution is ErrInterviewConflict.
	Cancel(ctx context.Context, req CancelRequest) (ExecutionFacts, error)
}

// CandidateProgress is the progress summary Interview Runtime marks as visible to the candidate.
type CandidateProgress struct {
	// Phase is the current phase label.
	Phase string
	// Connection is connected, reconnecting or finalizing.
	Connection string
}

// InterviewProgress is the sibling Interview Runtime contract for candidate-safe progress.
type InterviewProgress interface {
	// Current returns the candidate-safe progress of the execution, or nil when runtime has none.
	Current(ctx context.Context, executionID string) (*CandidateProgress, error)
}
