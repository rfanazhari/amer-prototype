package execution_test

import (
	"testing"
	"time"

	"github.com/rfanazhari/amer-prototype/domain/aggregate/execution"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fixtureNow returns a reproducible UTC millisecond-precision instant.
func fixtureNow() time.Time { return time.Date(2026, 11, 2, 9, 0, 0, 123000000, time.UTC) }

// fixtureBinding returns a valid two-field binding.
func fixtureBinding() execution.Binding {
	return execution.Binding{InstanceID: "inst-1", RevisionID: "rev-1"}
}

// fixtureExecution creates a CREATED execution or fails the test.
func fixtureExecution(t *testing.T) *execution.Execution {
	t.Helper()
	e, _, _, err := execution.NewExecution("exec-1", fixtureBinding(), fixtureNow())
	require.NoError(t, err)
	return e
}

// fixtureRunning creates a CREATED execution then activates it to RUNNING.
func fixtureRunning(t *testing.T) *execution.Execution {
	t.Helper()
	e := fixtureExecution(t)
	_, _, err := e.Activate(1, "actor-1", "op-1", 1, nil, fixtureNow())
	require.NoError(t, err)
	return e
}

// TestNewExecution covers creation validation and the initial CREATED state.
func TestNewExecution(t *testing.T) {
	// newCase is one row in the NewExecution table-driven test.
	type newCase struct {
		name    string
		id      string
		binding execution.Binding
		now     time.Time
		wantErr bool
	}
	cases := []newCase{
		{
			name:    "valid",
			id:      "exec-1",
			binding: fixtureBinding(),
			now:     fixtureNow(),
			wantErr: false,
		},
		{
			name:    "empty id",
			id:      "",
			binding: fixtureBinding(),
			now:     fixtureNow(),
			wantErr: true,
		},
		{
			name:    "empty instance id",
			id:      "exec-1",
			binding: execution.Binding{InstanceID: "", RevisionID: "rev-1"},
			now:     fixtureNow(),
			wantErr: true,
		},
		{
			name:    "empty revision id",
			id:      "exec-1",
			binding: execution.Binding{InstanceID: "inst-1", RevisionID: ""},
			now:     fixtureNow(),
			wantErr: true,
		},
		{
			name:    "zero time",
			id:      "exec-1",
			binding: fixtureBinding(),
			now:     time.Time{},
			wantErr: true,
		},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			e, s, fact, err := execution.NewExecution(tt.id, tt.binding, tt.now)
			if tt.wantErr {
				require.Error(t, err)
				assert.Nil(t, e)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, e)
			assert.Equal(t, execution.StateCreated, s.State)
			assert.Equal(t, int64(1), s.StateVersion)
			assert.Nil(t, s.ActivatedAt)
			assert.Nil(t, s.TerminalAt)
			assert.Nil(t, s.ProgressRef)
			assert.Equal(t, "", s.TerminalReason)
			assert.Equal(t, execution.StateCreated, fact.ToState)
			assert.Equal(t, int64(1), fact.StateVersion)
		})
	}
}

// TestRestoreExecution covers valid restoration and rejection of unknown states.
func TestRestoreExecution(t *testing.T) {
	now := fixtureNow()
	activatedAt := now.Add(time.Second)
	terminalAt := now.Add(2 * time.Second)
	progress := execution.ProgressRef("prog-1")

	// restoreCase is one row in the RestoreExecution table-driven test.
	type restoreCase struct {
		name    string
		snap    execution.Snapshot
		wantErr bool
	}
	cases := []restoreCase{
		{
			name: "valid CREATED",
			snap: execution.Snapshot{
				ID: "exec-1", Binding: fixtureBinding(), State: execution.StateCreated,
				StateVersion: 1, CreatedAt: now, UpdatedAt: now,
			},
			wantErr: false,
		},
		{
			name: "valid RUNNING",
			snap: execution.Snapshot{
				ID: "exec-1", Binding: fixtureBinding(), State: execution.StateRunning,
				StateVersion: 2, ActivatedAt: &activatedAt, CreatedAt: now, UpdatedAt: activatedAt,
			},
			wantErr: false,
		},
		{
			name: "valid COMPLETED",
			snap: execution.Snapshot{
				ID: "exec-1", Binding: fixtureBinding(), State: execution.StateCompleted,
				StateVersion: 3, ActivatedAt: &activatedAt, TerminalAt: &terminalAt,
				ProgressRef: &progress, CreatedAt: now, UpdatedAt: terminalAt,
			},
			wantErr: false,
		},
		{
			name: "valid FAILED",
			snap: execution.Snapshot{
				ID: "exec-1", Binding: fixtureBinding(), State: execution.StateFailed,
				StateVersion: 2, TerminalAt: &terminalAt, TerminalReason: "provider_error",
				CreatedAt: now, UpdatedAt: terminalAt,
			},
			wantErr: false,
		},
		{
			name: "valid CANCELLED",
			snap: execution.Snapshot{
				ID: "exec-1", Binding: fixtureBinding(), State: execution.StateCancelled,
				StateVersion: 2, TerminalAt: &terminalAt, TerminalReason: "access_revoked",
				CreatedAt: now, UpdatedAt: terminalAt,
			},
			wantErr: false,
		},
		{
			name: "WAITING_INPUT rejected",
			snap: execution.Snapshot{
				ID: "exec-1", Binding: fixtureBinding(), State: execution.State("WAITING_INPUT"),
				StateVersion: 1, CreatedAt: now, UpdatedAt: now,
			},
			wantErr: true,
		},
		{
			name: "ACTIVATING rejected",
			snap: execution.Snapshot{
				ID: "exec-1", Binding: fixtureBinding(), State: execution.State("ACTIVATING"),
				StateVersion: 1, CreatedAt: now, UpdatedAt: now,
			},
			wantErr: true,
		},
		{
			name: "RECOVERING rejected",
			snap: execution.Snapshot{
				ID: "exec-1", Binding: fixtureBinding(), State: execution.State("RECOVERING"),
				StateVersion: 1, CreatedAt: now, UpdatedAt: now,
			},
			wantErr: true,
		},
		{
			name: "zero version rejected",
			snap: execution.Snapshot{
				ID: "exec-1", Binding: fixtureBinding(), State: execution.StateCreated,
				StateVersion: 0, CreatedAt: now, UpdatedAt: now,
			},
			wantErr: true,
		},
		{
			name: "empty id rejected",
			snap: execution.Snapshot{
				ID: "", Binding: fixtureBinding(), State: execution.StateCreated,
				StateVersion: 1, CreatedAt: now, UpdatedAt: now,
			},
			wantErr: true,
		},
		{
			name: "COMPLETED without progress rejected",
			snap: execution.Snapshot{
				ID: "exec-1", Binding: fixtureBinding(), State: execution.StateCompleted,
				StateVersion: 3, ActivatedAt: &activatedAt, TerminalAt: &terminalAt,
				ProgressRef: nil, CreatedAt: now, UpdatedAt: terminalAt,
			},
			wantErr: true,
		},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			e, err := execution.RestoreExecution(tt.snap)
			if tt.wantErr {
				require.Error(t, err)
				assert.ErrorIs(t, err, execution.ErrInvalidExecution)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, e)
		})
	}
}

// TestActivate covers CREATED→RUNNING transitions and rejection of illegal source states.
func TestActivate(t *testing.T) {
	now := fixtureNow()

	// activateCase is one row in the Activate table-driven test.
	type activateCase struct {
		name      string
		state     execution.State
		version   int64
		gen       int64
		wantErr   bool
		wantErrIs error
	}
	cases := []activateCase{
		{name: "CREATED to RUNNING", state: execution.StateCreated, version: 1, gen: 1, wantErr: false},
		{name: "RUNNING already", state: execution.StateRunning, version: 1, gen: 1, wantErr: true, wantErrIs: execution.ErrInvalidTransition},
		{name: "COMPLETED terminal", state: execution.StateCompleted, version: 1, gen: 1, wantErr: true, wantErrIs: execution.ErrInvalidTransition},
		{name: "FAILED terminal", state: execution.StateFailed, version: 1, gen: 1, wantErr: true, wantErrIs: execution.ErrInvalidTransition},
		{name: "CANCELLED terminal", state: execution.StateCancelled, version: 1, gen: 1, wantErr: true, wantErrIs: execution.ErrInvalidTransition},
		{name: "stale version", state: execution.StateCreated, version: 99, gen: 1, wantErr: true, wantErrIs: execution.ErrStaleVersion},
	}

	activatedAt := now.Add(time.Second)
	terminalAt := now.Add(2 * time.Second)
	progress := execution.ProgressRef("prog-1")

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			var snap execution.Snapshot
			switch tt.state {
			case execution.StateCreated:
				snap = execution.Snapshot{ID: "exec-1", Binding: fixtureBinding(), State: execution.StateCreated, StateVersion: 1, CreatedAt: now, UpdatedAt: now}
			case execution.StateRunning:
				snap = execution.Snapshot{ID: "exec-1", Binding: fixtureBinding(), State: execution.StateRunning, StateVersion: 2, ActivatedAt: &activatedAt, CreatedAt: now, UpdatedAt: activatedAt}
			case execution.StateCompleted:
				snap = execution.Snapshot{ID: "exec-1", Binding: fixtureBinding(), State: execution.StateCompleted, StateVersion: 3, ActivatedAt: &activatedAt, TerminalAt: &terminalAt, ProgressRef: &progress, CreatedAt: now, UpdatedAt: terminalAt}
			case execution.StateFailed:
				snap = execution.Snapshot{ID: "exec-1", Binding: fixtureBinding(), State: execution.StateFailed, StateVersion: 2, TerminalAt: &terminalAt, TerminalReason: "err", CreatedAt: now, UpdatedAt: terminalAt}
			case execution.StateCancelled:
				snap = execution.Snapshot{ID: "exec-1", Binding: fixtureBinding(), State: execution.StateCancelled, StateVersion: 2, TerminalAt: &terminalAt, TerminalReason: "cancelled", CreatedAt: now, UpdatedAt: terminalAt}
			}
			e, err := execution.RestoreExecution(snap)
			require.NoError(t, err)

			_, _, err = e.Activate(tt.version, "actor", "op-1", tt.gen, nil, now)
			if tt.wantErr {
				require.Error(t, err)
				if tt.wantErrIs != nil {
					assert.ErrorIs(t, err, tt.wantErrIs)
				}
				return
			}
			require.NoError(t, err)
			s := e.Snapshot()
			assert.Equal(t, execution.StateRunning, s.State)
			assert.Equal(t, int64(2), s.StateVersion)
			require.NotNil(t, s.ActivatedAt)
		})
	}
}

// TestTerminalRejectsAllTransitions verifies that completed, failed, and cancelled executions
// reject activate, progress commit, complete, fail, and cancel.
func TestTerminalRejectsAllTransitions(t *testing.T) {
	now := fixtureNow()
	terminalAt := now.Add(time.Second)
	activatedAt := now.Add(500 * time.Millisecond)
	progress := execution.ProgressRef("prog-1")

	completedSnap := execution.Snapshot{
		ID: "exec-1", Binding: fixtureBinding(), State: execution.StateCompleted,
		StateVersion: 3, ActivatedAt: &activatedAt, TerminalAt: &terminalAt,
		ProgressRef: &progress, CreatedAt: now, UpdatedAt: terminalAt,
	}
	failedSnap := execution.Snapshot{
		ID: "exec-1", Binding: fixtureBinding(), State: execution.StateFailed,
		StateVersion: 2, TerminalAt: &terminalAt, TerminalReason: "err", CreatedAt: now, UpdatedAt: terminalAt,
	}
	cancelledSnap := execution.Snapshot{
		ID: "exec-1", Binding: fixtureBinding(), State: execution.StateCancelled,
		StateVersion: 2, TerminalAt: &terminalAt, TerminalReason: "revoked", CreatedAt: now, UpdatedAt: terminalAt,
	}

	for _, snap := range []execution.Snapshot{completedSnap, failedSnap, cancelledSnap} {
		e, err := execution.RestoreExecution(snap)
		require.NoError(t, err)

		t.Run(string(snap.State)+"/activate", func(t *testing.T) {
			_, _, err := e.Activate(snap.StateVersion, "actor", "op", 1, nil, now)
			assert.ErrorIs(t, err, execution.ErrInvalidTransition)
		})
		t.Run(string(snap.State)+"/commitProgress", func(t *testing.T) {
			_, err := e.CommitProgress("prog-2", snap.StateVersion, 1, nil, now)
			assert.ErrorIs(t, err, execution.ErrInvalidTransition)
		})
		t.Run(string(snap.State)+"/complete", func(t *testing.T) {
			_, _, err := e.Complete("prog-2", snap.StateVersion, 1, nil, "actor", "op", now)
			assert.ErrorIs(t, err, execution.ErrInvalidTransition)
		})
		t.Run(string(snap.State)+"/fail", func(t *testing.T) {
			_, _, err := e.Fail("reason", snap.StateVersion, 1, nil, "actor", "op", now)
			assert.ErrorIs(t, err, execution.ErrInvalidTransition)
		})
		t.Run(string(snap.State)+"/cancel", func(t *testing.T) {
			_, _, err := e.Cancel("reason", "actor", now)
			assert.ErrorIs(t, err, execution.ErrInvalidTransition)
		})
	}
}

// TestCompleteRequiresProgressRef verifies that Complete fails without a progress reference.
func TestCompleteRequiresProgressRef(t *testing.T) {
	e := fixtureRunning(t)
	_, _, err := e.Complete("", 2, 1, nil, "actor", "op-1", fixtureNow())
	assert.ErrorIs(t, err, execution.ErrProgressRequired)
}

// TestAttemptFencing covers stale generation and deadline boundary.
func TestAttemptFencing(t *testing.T) {
	now := fixtureNow()
	future := now.Add(time.Hour)
	exactNow := now

	// fenceCase is one row in the attempt fencing table-driven test.
	type fenceCase struct {
		name      string
		gen       int64
		deadline  *time.Time
		wantErr   bool
		wantErrIs error
	}
	cases := []fenceCase{
		{name: "valid gen and no deadline", gen: 1, deadline: nil, wantErr: false},
		{name: "valid gen and future deadline", gen: 1, deadline: &future, wantErr: false},
		{name: "stale gen", gen: 0, deadline: nil, wantErr: true, wantErrIs: execution.ErrStaleAttempt},
		{name: "deadline equal to now", gen: 1, deadline: &exactNow, wantErr: true, wantErrIs: execution.ErrStaleAttempt},
		{name: "deadline before now", gen: 1, deadline: func() *time.Time { t := now.Add(-time.Millisecond); return &t }(), wantErr: true, wantErrIs: execution.ErrStaleAttempt},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			// Use a fresh RUNNING execution for each case; gen must be >= 1 (current gen after activation).
			snap := execution.Snapshot{
				ID: "exec-1", Binding: fixtureBinding(), State: execution.StateRunning,
				StateVersion: 2, CurrentGeneration: 1,
				ActivatedAt: func() *time.Time { v := now; return &v }(),
				CreatedAt: now, UpdatedAt: now,
			}
			e, err := execution.RestoreExecution(snap)
			require.NoError(t, err)

			_, err = e.CommitProgress("prog-ref", 2, tt.gen, tt.deadline, now)
			if tt.wantErr {
				require.Error(t, err)
				if tt.wantErrIs != nil {
					assert.ErrorIs(t, err, tt.wantErrIs)
				}
				return
			}
			require.NoError(t, err)
		})
	}
}

// TestBindingEquality verifies that Binding.Equal compares only InstanceID and RevisionID.
func TestBindingEquality(t *testing.T) {
	a := execution.Binding{InstanceID: "inst-1", RevisionID: "rev-1"}
	b := execution.Binding{InstanceID: "inst-1", RevisionID: "rev-1"}
	c := execution.Binding{InstanceID: "inst-1", RevisionID: "rev-2"}
	d := execution.Binding{InstanceID: "inst-2", RevisionID: "rev-1"}

	assert.True(t, a.Equal(b), "same binding should be equal")
	assert.False(t, a.Equal(c), "different revision should not be equal")
	assert.False(t, a.Equal(d), "different instance should not be equal")
}

// TestSnapshotCopiesTimePointers verifies that Snapshot does not alias internal time values.
func TestSnapshotCopiesTimePointers(t *testing.T) {
	now := fixtureNow()
	e := fixtureRunning(t)
	s1 := e.Snapshot()
	require.NotNil(t, s1.ActivatedAt)

	// Mutate the returned pointer — the aggregate's internal state must be unaffected.
	*s1.ActivatedAt = now.Add(24 * time.Hour)
	s2 := e.Snapshot()
	require.NotNil(t, s2.ActivatedAt)
	assert.NotEqual(t, *s1.ActivatedAt, *s2.ActivatedAt, "mutating the snapshot must not change the aggregate")
}

// TestFailFromCreated verifies that Fail works from CREATED state.
func TestFailFromCreated(t *testing.T) {
	e := fixtureExecution(t)
	now := fixtureNow()
	snap, fact, err := e.Fail("provider_error", 1, 1, nil, "actor", "op-1", now)
	require.NoError(t, err)
	assert.Equal(t, execution.StateFailed, snap.State)
	assert.Equal(t, "provider_error", snap.TerminalReason)
	assert.Equal(t, execution.StateFailed, fact.ToState)
	require.NotNil(t, fact.FromState)
	assert.Equal(t, execution.StateCreated, *fact.FromState)
}

// TestCancelFromRunning verifies that Cancel works from RUNNING state.
func TestCancelFromRunning(t *testing.T) {
	e := fixtureRunning(t)
	now := fixtureNow()
	snap, fact, err := e.Cancel("access_revoked", "actor", now)
	require.NoError(t, err)
	assert.Equal(t, execution.StateCancelled, snap.State)
	assert.Equal(t, "access_revoked", snap.TerminalReason)
	assert.Equal(t, execution.StateCancelled, fact.ToState)
}

// TestCompleteFromRunning verifies the RUNNING→COMPLETED happy path.
func TestCompleteFromRunning(t *testing.T) {
	e := fixtureRunning(t)
	now := fixtureNow()
	snap, fact, err := e.Complete("prog-ref-1", 2, 1, nil, "actor", "op-1", now)
	require.NoError(t, err)
	assert.Equal(t, execution.StateCompleted, snap.State)
	require.NotNil(t, snap.ProgressRef)
	assert.Equal(t, execution.ProgressRef("prog-ref-1"), *snap.ProgressRef)
	assert.Equal(t, execution.StateCompleted, fact.ToState)
}
