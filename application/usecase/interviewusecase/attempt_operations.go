package interviewusecase

import (
	"context"
	"errors"

	"github.com/rfanazhari/amer-prototype/application/service"
)

var (
	// ErrAttemptInvalid reports a session-attempt or readiness input the operation cannot act on.
	ErrAttemptInvalid = errors.New("invalid interview session attempt")
	// ErrAttemptConflict reports a stale version, wrong controller, reused key with another intent or exhausted budget.
	ErrAttemptConflict = errors.New("interview session attempt conflict")
	// ErrAttemptNotReady reports readiness for an attempt that is not current or a session blocked by a technical loss.
	ErrAttemptNotReady = errors.New("interview session attempt not ready")
	// ErrProviderUnavailable reports a provider failure before a durable attempt result exists.
	ErrProviderUnavailable = errors.New("interview voice provider unavailable")
)

const (
	// IntentActivate opens the first session of an execution.
	IntentActivate = "activate"
	// IntentReconnect opens a replacement session after a lost connection.
	IntentReconnect = "reconnect"
	// ConditionPending means the provider identity or browser readiness is still missing.
	ConditionPending = "pending"
	// ConditionReady means lifecycle, provider identity and browser readiness agree for the attempt.
	ConditionReady = "ready"
)

// AttemptInput carries one candidate session-attempt request.
type AttemptInput struct {
	// ExecutionID is the execution in the path.
	ExecutionID string
	// ControllerID is the tab id from the request body.
	ControllerID string
	// IdempotencyKey scopes the replay to the candidate session and execution.
	IdempotencyKey string
	// Intent is activate or reconnect.
	Intent string
	// SDP is the browser offer.
	SDP string
	// ExpectedVersion is the execution version the lifecycle must still hold.
	ExpectedVersion int64
}

// AttemptResult is the attempt a call created or replayed.
type AttemptResult struct {
	// AttemptID is the lifecycle attempt identity.
	AttemptID string
	// SessionID is the local voice-session identity.
	SessionID string
	// SDP is the provider answer.
	SDP string
	// ExecutionVersion is the version after the durable attempt is recorded.
	ExecutionVersion int64
	// SessionCondition is pending or ready.
	SessionCondition string
	// Replayed is true when the original result was returned.
	Replayed bool
}

// AttemptOperation creates or replays one voice attempt.
type AttemptOperation interface {
	// Execute creates the attempt, or returns the stored result for the same key and intent.
	Execute(ctx context.Context, in AttemptInput) (*AttemptResult, error)
}

// ReadyInput carries browser readiness for one attempt.
type ReadyInput struct {
	// ExecutionID is the execution in the path.
	ExecutionID string
	// AttemptID is the attempt in the path.
	AttemptID string
	// ControllerID is the tab id from the request body.
	ControllerID string
	// ExpectedVersion is the execution version the lifecycle must still hold.
	ExpectedVersion int64
}

// ReadyResult reports the attempt after readiness was recorded.
type ReadyResult struct {
	// AttemptID is the attempt that was marked.
	AttemptID string
	// SessionID is the local voice-session identity.
	SessionID string
	// ExecutionVersion is the lifecycle version observed.
	ExecutionVersion int64
	// SessionCondition is pending or ready.
	SessionCondition string
}

// ReadyOperation records browser media readiness for the current attempt only.
type ReadyOperation interface {
	// Execute sets the readiness flag. It never changes coverage or the stage.
	Execute(ctx context.Context, in ReadyInput) (*ReadyResult, error)
}

// SessionStarter opens a provider session under lifecycle authority; Workflow implements it.
type SessionStarter interface {
	// Activate opens the first session of the execution.
	Activate(ctx context.Context, req SessionRequest) (service.InterviewVoiceCreated, error)
	// Reconnect opens a replacement session within the stored recovery budget.
	Reconnect(ctx context.Context, req SessionRequest) (service.InterviewVoiceCreated, error)
}

var _ SessionStarter = Workflow{}
