package executionusecase

import (
	"context"
	"time"

	"github.com/rfanazhari/amer-prototype/application/dto"
)

// StartInput carries the caller-supplied fields for an interview execution start.
type StartInput struct {
	// AssignmentID is the interview assignment used as the execution scope.
	AssignmentID string
	// RequestKey is the opaque idempotency key supplied by the caller.
	RequestKey string
	// ActorRef is the opaque identifier of the caller initiating the start.
	ActorRef string
	// InputRefs is the ordered set of opaque input reference tokens included in the intent fingerprint.
	InputRefs []string
}

// AdmitInput carries the caller-supplied fields for a non-interview execution admission.
type AdmitInput struct {
	// Scope is the opaque scope identifier for this execution.
	Scope string
	// RequestKey is the opaque idempotency key supplied by the caller.
	RequestKey string
	// ActorRef is the opaque identifier of the caller requesting admission.
	ActorRef string
	// InstanceID is the agent instance identity resolved by the caller.
	InstanceID string
	// RevisionID is the published revision identity resolved by the caller.
	RevisionID string
	// InputRefs is the ordered set of opaque input reference tokens included in the intent fingerprint.
	InputRefs []string
}

// ActivateInput carries the caller-supplied fields for an activation attempt.
type ActivateInput struct {
	// ExecutionID is the execution to activate.
	ExecutionID string
	// RequestKey is the opaque idempotency key supplied by the caller.
	RequestKey string
	// ActorRef is the opaque identifier of the runtime performing the activation.
	ActorRef string
	// ExpectedVersion is the state version the caller last observed.
	ExpectedVersion int64
	// OperationKey is the opaque key identifying the activation operation.
	OperationKey string
	// Generation is the attempt authority generation for fencing.
	Generation int64
	// Deadline, when set, is the instant at or after which the attempt is too late.
	Deadline *time.Time
}

// RecordAttemptInput carries the caller-supplied fields for recording a new attempt.
type RecordAttemptInput struct {
	// ExecutionID is the execution receiving the new attempt.
	ExecutionID string
	// RequestKey is the opaque idempotency key supplied by the caller.
	RequestKey string
	// ActorRef is the opaque identifier of the runtime recording the attempt.
	ActorRef string
	// ExpectedVersion is the state version the caller last observed.
	ExpectedVersion int64
	// OperationKey is the opaque key identifying the operation receiving the attempt.
	OperationKey string
	// Generation is the new attempt authority generation, which must exceed the current one.
	Generation int64
}

// CommitProgressInput carries the caller-supplied fields for recording committed progress.
type CommitProgressInput struct {
	// ExecutionID is the execution to update.
	ExecutionID string
	// RequestKey is the opaque idempotency key supplied by the caller.
	RequestKey string
	// ActorRef is the opaque identifier of the runtime committing progress.
	ActorRef string
	// ExpectedVersion is the state version the caller last observed.
	ExpectedVersion int64
	// OperationKey is the opaque key for the operation whose progress is being committed.
	OperationKey string
	// Generation is the attempt generation asserting authority to write progress.
	Generation int64
	// ProgressRef is the opaque token identifying the committed progress in the sibling store.
	ProgressRef string
	// Deadline, when set, is the instant at or after which the attempt is too late.
	Deadline *time.Time
}

// CompleteInput carries the caller-supplied fields for completing an execution.
type CompleteInput struct {
	// ExecutionID is the execution to complete.
	ExecutionID string
	// RequestKey is the opaque idempotency key supplied by the caller.
	RequestKey string
	// ActorRef is the opaque identifier of the runtime requesting completion.
	ActorRef string
	// ExpectedVersion is the state version the caller last observed.
	ExpectedVersion int64
	// OperationKey is the opaque key for the operation asserting completion.
	OperationKey string
	// Generation is the attempt generation asserting authority to complete.
	Generation int64
	// ProgressRef is the opaque token identifying the committed progress in the sibling store.
	ProgressRef string
	// Deadline, when set, is the instant at or after which the attempt is too late.
	Deadline *time.Time
}

// FailInput carries the caller-supplied fields for failing an execution.
type FailInput struct {
	// ExecutionID is the execution to fail.
	ExecutionID string
	// RequestKey is the opaque idempotency key supplied by the caller.
	RequestKey string
	// ActorRef is the opaque identifier of the runtime failing the execution.
	ActorRef string
	// ExpectedVersion is the state version the caller last observed.
	ExpectedVersion int64
	// OperationKey is the opaque key for the operation that failed.
	OperationKey string
	// Generation is the attempt generation asserting authority to fail.
	Generation int64
	// Reason is the opaque reason token stored on the terminal row.
	Reason string
	// Deadline, when set, is the instant at or after which the attempt is too late.
	Deadline *time.Time
}

// CancelInput carries the caller-supplied fields for cancelling an execution.
type CancelInput struct {
	// ExecutionID is the execution to cancel.
	ExecutionID string
	// RequestKey is the opaque idempotency key supplied by the caller.
	RequestKey string
	// ActorRef is the opaque identifier of the caller requesting cancellation.
	ActorRef string
	// ExpectedVersion is the state version the caller last observed.
	ExpectedVersion int64
	// Reason is the opaque reason token stored on the terminal row.
	Reason string
}

// CancelNonterminalInput carries the caller-supplied fields for cancelling a nonterminal execution
// when assignment access has ended.
type CancelNonterminalInput struct {
	// AssignmentID is the assignment whose access has ended, used to locate and lock the execution.
	AssignmentID string
	// ActorRef is the opaque identifier of the caller committing the access end.
	ActorRef string
	// Reason is the opaque reason token stored on the terminal row when cancellation occurs.
	Reason string
}

// StartOperation starts or finds an interview execution for the given assignment.
// The same key and fingerprint return the original execution. A conflicting fingerprint is rejected.
type StartOperation interface {
	// Execute starts or returns the existing execution for the assignment.
	Execute(ctx context.Context, input StartInput) (*dto.ExecutionView, error)
}

// AdmitOperation admits a non-interview execution bound to a caller-resolved revision.
// It does not write an interview pin and does not read an assignment.
type AdmitOperation interface {
	// Execute admits a new execution or returns the existing one for the scope and key.
	Execute(ctx context.Context, input AdmitInput) (*dto.ExecutionView, error)
}

// ActivateOperation moves a CREATED execution to RUNNING.
type ActivateOperation interface {
	// Execute activates the execution or records a failed attempt.
	Execute(ctx context.Context, input ActivateInput) (*dto.ExecutionView, error)
}

// RecordAttemptOperation records a new attempt authority and updates the current generation.
type RecordAttemptOperation interface {
	// Execute records the new attempt, replacing authority for the operation key.
	Execute(ctx context.Context, input RecordAttemptInput) (*dto.ExecutionView, error)
}

// CommitProgressOperation records an opaque progress reference while the execution is RUNNING.
type CommitProgressOperation interface {
	// Execute stores the progress reference without completing the execution.
	Execute(ctx context.Context, input CommitProgressInput) (*dto.ExecutionView, error)
}

// CompleteOperation moves a RUNNING execution to COMPLETED when committed progress is confirmed.
type CompleteOperation interface {
	// Execute completes the execution if the progress port confirms the reference is committed.
	Execute(ctx context.Context, input CompleteInput) (*dto.ExecutionView, error)
}

// FailOperation moves a CREATED or RUNNING execution to FAILED.
type FailOperation interface {
	// Execute fails the execution with the supplied reason token.
	Execute(ctx context.Context, input FailInput) (*dto.ExecutionView, error)
}

// CancelOperation moves a CREATED or RUNNING execution to CANCELLED.
type CancelOperation interface {
	// Execute cancels the execution with the supplied reason token.
	Execute(ctx context.Context, input CancelInput) (*dto.ExecutionView, error)
}

// CancelNonterminalOperation cancels a nonterminal execution when assignment access has ended.
// It locks the assignment before the execution and leaves an already terminal row unchanged.
type CancelNonterminalOperation interface {
	// Execute cancels the execution if it is not yet terminal; a terminal row is returned unchanged.
	Execute(ctx context.Context, input CancelNonterminalInput) error
}
