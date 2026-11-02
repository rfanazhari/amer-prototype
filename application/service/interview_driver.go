package service

import "context"

// InterviewVoiceSession correlates provider observations with local execution and attempt authority.
// ProviderSessionID is empty only before create succeeds. SessionID is the local session identity.
type InterviewVoiceSession struct {
	ExecutionID       string
	AttemptID         string
	SessionID         string
	ProviderSessionID string
}

// InterviewVoiceCreate carries an SDP offer and application-built instructions for one session.
// The caller supplies normalized SDP. Credentials belong to the adapter, never this request.
type InterviewVoiceCreate struct {
	Session      InterviewVoiceSession
	SDP          string
	Instructions string
}

// InterviewVoiceCreated retains the provider identity and SDP answer required for readiness.
type InterviewVoiceCreated struct {
	// ExecutionVersion is filled by workflow after the result transaction commits.
	ExecutionVersion int64
	Session          InterviewVoiceSession
	SDP              string
}

// InterviewVoiceObservation is an opaque provider event or transport observation.
// Kind is event, connected or disconnected. EventType and Payload have no finality semantics;
// only the separately reviewed classifier may turn them into evidence. Payload is caller-owned.
// Session carries the attempt identity so a superseded observation can be rejected.
type InterviewVoiceObservation struct {
	Session   InterviewVoiceSession
	Kind      string
	EventType string
	Payload   []byte
}

// InterviewVoiceContext contains application-built context from committed progress only.
// Text excludes the salary reference and amount; the driver does not build HR instructions.
type InterviewVoiceContext struct {
	Session InterviewVoiceSession
	Text    string
}

// InterviewVoiceControl carries an opaque transport command, without deciding coverage or stages.
type InterviewVoiceControl struct {
	Session InterviewVoiceSession
	Command string
	Payload []byte
}

// InterviewDriver supplies voice transport observations without interpreting evidence finality.
// Every call honors context cancellation. Attach resources live until Close or context cancellation.
type InterviewDriver interface {
	// Create opens one provider session and returns its retained id and SDP answer.
	Create(ctx context.Context, req InterviewVoiceCreate) (InterviewVoiceCreated, error)
	// Attach opens backend control for an existing provider session.
	Attach(ctx context.Context, session InterviewVoiceSession) error
	// Observe waits for the next opaque event or transport change; cancellation ends the wait.
	Observe(ctx context.Context, session InterviewVoiceSession) (InterviewVoiceObservation, error)
	// SupplyContext sends application-built instructions and committed recovery context.
	SupplyContext(ctx context.Context, req InterviewVoiceContext) error
	// Control sends a transport command without mutating interview progress.
	Control(ctx context.Context, req InterviewVoiceControl) error
	// Close releases owned resources and attempts provider close; failure is returned and never
	// implies that a committed handoff can be rolled back.
	Close(ctx context.Context, session InterviewVoiceSession) error
}
