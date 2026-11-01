package execution

import "time"

// State is one of the five recognized lifecycle positions for an execution.
type State string

const (
	// StateCreated is the accepted-but-not-activated position.
	StateCreated State = "CREATED"
	// StateRunning means activation succeeded and agent work has not ended.
	StateRunning State = "RUNNING"
	// StateCompleted is a terminal outcome with committed progress.
	StateCompleted State = "COMPLETED"
	// StateFailed is a terminal outcome for an irrecoverable failure.
	StateFailed State = "FAILED"
	// StateCancelled is a terminal outcome when access was withdrawn or work was abandoned.
	StateCancelled State = "CANCELLED"
)

// Binding is the pair of identifiers copied at admission and never changed by later publication.
type Binding struct {
	// InstanceID is the agent instance identity copied at admission.
	InstanceID string
	// RevisionID is the published revision identity copied at admission.
	RevisionID string
}

// Equal reports whether two bindings represent the same instance and revision,
// ignoring any newer active revision that may exist on the agent instance.
func (b Binding) Equal(other Binding) bool {
	return b.InstanceID == other.InstanceID && b.RevisionID == other.RevisionID
}

// valid reports that both identifiers are non-empty.
func (b Binding) valid() bool { return b.InstanceID != "" && b.RevisionID != "" }

// ProgressRef is a non-empty opaque token identifying committed progress in a sibling store.
type ProgressRef string

// TransitionFact carries the data the repository must append as an execution history row.
type TransitionFact struct {
	// FromState is the previous state, nil for the initial CREATED transition.
	FromState *State
	// ToState is the state after the transition.
	ToState State
	// StateVersion is the version number assigned to ToState.
	StateVersion int64
	// ActorRef is the opaque identifier of the caller who caused the transition.
	ActorRef string
	// OperationKey is the opaque key for the attempt that caused the transition, when relevant.
	OperationKey string
	// Reason is the terminal reason supplied for FAILED or CANCELLED transitions.
	Reason string
	// OccurredAt is the trusted instant at which the transition was committed.
	OccurredAt time.Time
}

// Snapshot is a detached copy of the current execution state with independent time pointers.
type Snapshot struct {
	// ID is the stable execution identity.
	ID string
	// Binding is the immutable instance and revision copied at admission.
	Binding Binding
	// State is the current lifecycle position.
	State State
	// StateVersion is the monotonically increasing version incremented on each transition.
	StateVersion int64
	// CurrentGeneration is the current attempt authority generation.
	CurrentGeneration int64
	// ProgressRef is the opaque token for committed progress, set on CommitProgress and Complete.
	ProgressRef *ProgressRef
	// ActivatedAt is the trusted instant at which the execution moved to RUNNING, nil before activation.
	ActivatedAt *time.Time
	// TerminalAt is the trusted instant of the terminal transition, nil before termination.
	TerminalAt *time.Time
	// TerminalReason is the token supplied at FAILED or CANCELLED, empty for non-terminal states.
	TerminalReason string
	// CreatedAt is the trusted instant at which the execution was first committed.
	CreatedAt time.Time
	// UpdatedAt is the trusted instant of the most recent mutation.
	UpdatedAt time.Time
}

// Execution is the aggregate root for a single agent run.
// It owns identity, binding, state, version, progress reference, and attempt authority.
type Execution struct {
	state Snapshot
}

// valid reports whether s is a recognized execution state.
func (s State) valid() bool {
	switch s {
	case StateCreated, StateRunning, StateCompleted, StateFailed, StateCancelled:
		return true
	}
	return false
}

// terminal reports whether the state is a terminal outcome that cannot be reopened.
func (s State) terminal() bool {
	return s == StateCompleted || s == StateFailed || s == StateCancelled
}

// consistent reports whether the state, version, time, and progress combination is coherent.
func (s Snapshot) consistent() bool {
	if s.ID == "" || !s.Binding.valid() || !s.State.valid() || s.StateVersion < 1 || s.CreatedAt.IsZero() || s.UpdatedAt.IsZero() {
		return false
	}
	switch s.State {
	case StateCreated:
		return s.ActivatedAt == nil && s.TerminalAt == nil && s.ProgressRef == nil && s.TerminalReason == ""
	case StateRunning:
		return s.ActivatedAt != nil && !s.ActivatedAt.Before(s.CreatedAt) && s.TerminalAt == nil && s.TerminalReason == ""
	case StateCompleted:
		return s.ActivatedAt != nil && s.TerminalAt != nil && !s.TerminalAt.Before(*s.ActivatedAt) &&
			s.ProgressRef != nil && *s.ProgressRef != "" && s.TerminalReason == ""
	case StateFailed, StateCancelled:
		return s.TerminalAt != nil && !s.TerminalAt.Before(s.CreatedAt) &&
			(s.ActivatedAt == nil || !s.TerminalAt.Before(*s.ActivatedAt)) && s.TerminalReason != ""
	}
	return false
}

// NewExecution creates a CREATED execution and returns its snapshot and initial transition fact.
// It rejects an empty id, an invalid binding, or a zero trusted instant.
func NewExecution(id string, binding Binding, now time.Time) (*Execution, Snapshot, TransitionFact, error) {
	if id == "" || !binding.valid() || now.IsZero() {
		return nil, Snapshot{}, TransitionFact{}, ErrInvalidExecution
	}
	s := Snapshot{
		ID: id, Binding: binding, State: StateCreated,
		StateVersion: 1, CreatedAt: now, UpdatedAt: now,
	}
	fact := TransitionFact{ToState: StateCreated, StateVersion: 1, OccurredAt: now}
	e := &Execution{state: s.detach()}
	return e, s.detach(), fact, nil
}

// RestoreExecution rebuilds the aggregate from a previously committed consistent snapshot.
// It rejects unknown states (WAITING_INPUT, ACTIVATING, RECOVERING, and any other non-five state),
// a version below 1, and any inconsistent time or progress combination.
func RestoreExecution(s Snapshot) (*Execution, error) {
	if !s.consistent() {
		return nil, ErrInvalidExecution
	}
	return &Execution{state: s.detach()}, nil
}

// Snapshot returns a detached copy of the current execution state.
// Time pointer fields are copied so callers cannot mutate the aggregate through them.
func (e *Execution) Snapshot() Snapshot { return e.state.detach() }

// fenceAttempt rejects a lower generation or a deadline that is not strictly ahead of trusted now.
func (e *Execution) fenceAttempt(gen int64, deadline *time.Time, now time.Time) error {
	if gen < e.state.CurrentGeneration {
		return ErrStaleAttempt
	}
	if deadline != nil && !now.Before(*deadline) {
		return ErrStaleAttempt
	}
	return nil
}

// Activate moves the execution from CREATED to RUNNING.
// It rejects any non-CREATED source state, a stale expected version, and stale fencing.
func (e *Execution) Activate(expectedVersion int64, actorRef, operationKey string, gen int64, deadline *time.Time, now time.Time) (Snapshot, TransitionFact, error) {
	if e.state.State.terminal() || e.state.State != StateCreated {
		return Snapshot{}, TransitionFact{}, ErrInvalidTransition
	}
	if e.state.StateVersion != expectedVersion {
		return Snapshot{}, TransitionFact{}, ErrStaleVersion
	}
	if err := e.fenceAttempt(gen, deadline, now); err != nil {
		return Snapshot{}, TransitionFact{}, err
	}
	from := e.state.State
	e.state.State = StateRunning
	e.state.StateVersion++
	e.state.CurrentGeneration = gen
	at := now
	e.state.ActivatedAt = &at
	e.state.UpdatedAt = now
	fact := TransitionFact{FromState: &from, ToState: StateRunning, StateVersion: e.state.StateVersion, ActorRef: actorRef, OperationKey: operationKey, OccurredAt: now}
	return e.state.detach(), fact, nil
}

// RejectActivation records a failed activation attempt without changing the execution state.
// It rejects any non-CREATED source state, a stale expected version, and stale fencing.
func (e *Execution) RejectActivation(expectedVersion int64, actorRef, operationKey, reason string, gen int64, deadline *time.Time, now time.Time) error {
	if e.state.State != StateCreated {
		return ErrInvalidTransition
	}
	if e.state.StateVersion != expectedVersion {
		return ErrStaleVersion
	}
	return e.fenceAttempt(gen, deadline, now)
}

// CommitProgress records an opaque progress reference while the execution is RUNNING.
// It rejects any non-RUNNING source state, a stale expected version, and stale fencing.
func (e *Execution) CommitProgress(ref ProgressRef, expectedVersion int64, gen int64, deadline *time.Time, now time.Time) (Snapshot, error) {
	if e.state.State.terminal() || e.state.State != StateRunning {
		return Snapshot{}, ErrInvalidTransition
	}
	if e.state.StateVersion != expectedVersion {
		return Snapshot{}, ErrStaleVersion
	}
	if err := e.fenceAttempt(gen, deadline, now); err != nil {
		return Snapshot{}, err
	}
	e.state.ProgressRef = &ref
	e.state.UpdatedAt = now
	return e.state.detach(), nil
}

// Complete moves the execution from RUNNING to COMPLETED using the supplied progress reference.
// It rejects an empty reference, any non-RUNNING source state, a stale expected version, and stale fencing.
func (e *Execution) Complete(ref ProgressRef, expectedVersion int64, gen int64, deadline *time.Time, actorRef, operationKey string, now time.Time) (Snapshot, TransitionFact, error) {
	if ref == "" {
		return Snapshot{}, TransitionFact{}, ErrProgressRequired
	}
	if e.state.State.terminal() || e.state.State != StateRunning {
		return Snapshot{}, TransitionFact{}, ErrInvalidTransition
	}
	if e.state.StateVersion != expectedVersion {
		return Snapshot{}, TransitionFact{}, ErrStaleVersion
	}
	if err := e.fenceAttempt(gen, deadline, now); err != nil {
		return Snapshot{}, TransitionFact{}, err
	}
	from := e.state.State
	e.state.ProgressRef = &ref
	e.state.State = StateCompleted
	e.state.StateVersion++
	at := now
	e.state.TerminalAt = &at
	e.state.UpdatedAt = now
	fact := TransitionFact{FromState: &from, ToState: StateCompleted, StateVersion: e.state.StateVersion, ActorRef: actorRef, OperationKey: operationKey, OccurredAt: now}
	return e.state.detach(), fact, nil
}

// Fail moves the execution from CREATED or RUNNING to FAILED.
// It rejects any terminal source state, an empty reason, a stale expected version, and stale fencing.
func (e *Execution) Fail(reason string, expectedVersion int64, gen int64, deadline *time.Time, actorRef, operationKey string, now time.Time) (Snapshot, TransitionFact, error) {
	if e.state.State.terminal() {
		return Snapshot{}, TransitionFact{}, ErrInvalidTransition
	}
	if e.state.State != StateCreated && e.state.State != StateRunning {
		return Snapshot{}, TransitionFact{}, ErrInvalidTransition
	}
	if reason == "" {
		return Snapshot{}, TransitionFact{}, ErrInvalidExecution
	}
	if e.state.StateVersion != expectedVersion {
		return Snapshot{}, TransitionFact{}, ErrStaleVersion
	}
	if err := e.fenceAttempt(gen, deadline, now); err != nil {
		return Snapshot{}, TransitionFact{}, err
	}
	from := e.state.State
	e.state.State = StateFailed
	e.state.StateVersion++
	at := now
	e.state.TerminalAt = &at
	e.state.TerminalReason = reason
	e.state.UpdatedAt = now
	fact := TransitionFact{FromState: &from, ToState: StateFailed, StateVersion: e.state.StateVersion, ActorRef: actorRef, OperationKey: operationKey, Reason: reason, OccurredAt: now}
	return e.state.detach(), fact, nil
}

// Cancel moves the execution from CREATED or RUNNING to CANCELLED.
// It rejects any terminal source state and an empty reason.
func (e *Execution) Cancel(reason string, actorRef string, now time.Time) (Snapshot, TransitionFact, error) {
	if e.state.State.terminal() {
		return Snapshot{}, TransitionFact{}, ErrInvalidTransition
	}
	if e.state.State != StateCreated && e.state.State != StateRunning {
		return Snapshot{}, TransitionFact{}, ErrInvalidTransition
	}
	if reason == "" {
		return Snapshot{}, TransitionFact{}, ErrInvalidExecution
	}
	from := e.state.State
	e.state.State = StateCancelled
	e.state.StateVersion++
	at := now
	e.state.TerminalAt = &at
	e.state.TerminalReason = reason
	e.state.UpdatedAt = now
	fact := TransitionFact{FromState: &from, ToState: StateCancelled, StateVersion: e.state.StateVersion, ActorRef: actorRef, Reason: reason, OccurredAt: now}
	return e.state.detach(), fact, nil
}

// detach copies time pointers to ensure callers cannot mutate the aggregate's internal state.
func (s Snapshot) detach() Snapshot {
	if s.ActivatedAt != nil {
		v := *s.ActivatedAt
		s.ActivatedAt = &v
	}
	if s.TerminalAt != nil {
		v := *s.TerminalAt
		s.TerminalAt = &v
	}
	if s.ProgressRef != nil {
		ref := *s.ProgressRef
		s.ProgressRef = &ref
	}
	return s
}
