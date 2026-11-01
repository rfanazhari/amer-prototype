package candidatelifecycle_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/rfanazhari/amer-prototype/application/repository/executionport"
	"github.com/rfanazhari/amer-prototype/application/service"
	"github.com/rfanazhari/amer-prototype/domain/aggregate/execution"
	"github.com/rfanazhari/amer-prototype/infrastructure/service/candidatelifecycle"
	"github.com/rfanazhari/amer-prototype/test/mocks"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

const (
	assignmentID = "11111111-1111-4111-8111-111111111111"
	executionID  = "22222222-2222-4222-8222-222222222222"
	consentID    = "33333333-3333-4333-8333-333333333333"
)

// fixture owns generated mocks for each execution boundary and a controlled clock.
type fixture struct {
	adapter     candidatelifecycle.Lifecycle
	assignments *mocks.AssignmentLock
	tx          *mocks.Transactor
	store       *mocks.ExecutionStore
	commands    *mocks.CommandStore
	now         time.Time
}

// setup creates the adapter with strict Mockery ports.
func setup(t *testing.T) fixture {
	t.Helper()
	f := fixture{assignments: mocks.NewAssignmentLock(t), tx: mocks.NewTransactor(t), store: mocks.NewExecutionStore(t), commands: mocks.NewCommandStore(t), now: time.Date(2026, 11, 2, 1, 0, 0, 0, time.UTC)}
	clock := mocks.NewClock(t)
	clock.On("Now").Return(f.now).Maybe()
	f.adapter = candidatelifecycle.Lifecycle{Assignments: f.assignments, Transactions: f.tx, Executions: f.store, Commands: f.commands, Clock: clock}
	return f
}

// snapshot restores a valid fixture execution in the requested lifecycle position.
func snapshot(t *testing.T, now time.Time, state execution.State) execution.Snapshot {
	t.Helper()
	e, _, _, err := execution.NewExecution(executionID, execution.Binding{InstanceID: "instance", RevisionID: "revision"}, now)
	require.NoError(t, err)
	if state == execution.StateCreated {
		return e.Snapshot()
	}
	snap, _, err := e.Activate(1, "runtime", "op", 1, nil, now.Add(time.Minute))
	require.NoError(t, err)
	if state == execution.StateRunning {
		return snap
	}
	e, err = execution.RestoreExecution(snap)
	require.NoError(t, err)
	if state == execution.StateCompleted {
		snap, _, err = e.Complete("progress", 2, 1, nil, "runtime", "op", now.Add(2*time.Minute))
	} else {
		snap, _, err = e.Cancel(service.CancelReasonCandidateEarlyStop, "candidate-interview", now.Add(2*time.Minute))
	}
	require.NoError(t, err)
	return snap
}

// TestCommitOrFindReportsCommitAndReplay covers first commit and replay of a terminal execution.
func TestCommitOrFindReportsCommitAndReplay(t *testing.T) {
	type testCase struct {
		name     string
		existing bool
		wantErr  bool
	}
	for _, tt := range []testCase{{name: "first"}, {name: "lost response replay", existing: true}} {
		t.Run(tt.name, func(t *testing.T) {
			f := setup(t)
			f.assignments.On("WithAssignment", mock.Anything, assignmentID, mock.Anything).Return(func(ctx context.Context, _ string, fn func(context.Context, executionport.AssignmentRow) error) error {
				return fn(ctx, executionport.AssignmentRow{AssignmentID: assignmentID, InstanceID: "instance", RevisionID: "revision"})
			})
			var saved execution.Snapshot
			if tt.existing {
				saved = snapshot(t, f.now, execution.StateCompleted)
				f.store.On("FindByAssignment", mock.Anything, assignmentID).Return(saved, nil)
			} else {
				f.store.On("FindByAssignment", mock.Anything, assignmentID).Return(execution.Snapshot{}, executionport.ErrNotFound)
				f.store.On("InsertInterview", mock.Anything, assignmentID, mock.Anything, mock.Anything).Return(func(_ context.Context, _ string, s execution.Snapshot, _ execution.TransitionFact) error {
					saved = s
					return nil
				})
			}
			f.commands.On("FindOrInsert", mock.Anything, assignmentID, "start", "candidate-start", mock.Anything, mock.Anything).Return(func(_ context.Context, _, _, _, _, id string) (string, error) { return id, nil })
			result, err := f.adapter.CommitOrFind(context.Background(), service.CommitOrFindRequest{AssignmentID: assignmentID, ConsentID: consentID, IdempotencyKey: strings.Repeat("k", 128)})
			require.NoError(t, err)
			require.Equal(t, !tt.existing, result.Created)
			require.Equal(t, saved.ID, result.Execution.ID)
			require.Equal(t, assignmentID, result.Execution.AssignmentID)
			require.Equal(t, string(saved.State), result.Execution.State)
		})
	}
}

// TestFindReturnsDeadlineAndClassifiedErrors proves reads never start and deadline comes from persisted activation.
func TestFindReturnsDeadlineAndClassifiedErrors(t *testing.T) {
	type testCase struct {
		name    string
		state   execution.State
		source  error
		want    error
		wantErr bool
	}
	tests := []testCase{{name: "created", state: execution.StateCreated}, {name: "running", state: execution.StateRunning}, {name: "not found", source: executionport.ErrNotFound, want: service.ErrInterviewNotFound, wantErr: true}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := setup(t)
			var s execution.Snapshot
			if !tt.wantErr {
				s = snapshot(t, f.now, tt.state)
			}
			f.store.On("FindByAssignment", mock.Anything, assignmentID).Return(s, tt.source)
			got, err := f.adapter.Find(context.Background(), assignmentID)
			if tt.wantErr {
				require.ErrorIs(t, err, tt.want)
				return
			}
			require.NoError(t, err)
			require.Equal(t, assignmentID, got.AssignmentID)
			if tt.state == execution.StateRunning {
				require.Equal(t, f.now.Add(16*time.Minute), *got.HardDeadline)
				*got.HardDeadline = f.now
				require.Equal(t, f.now.Add(time.Minute), *s.ActivatedAt)
			} else {
				require.Nil(t, got.HardDeadline)
			}
		})
	}
}

// TestCancelUsesLifecycleAndRejectsTerminal checks cancellation under the execution lock, including terminal replay refusal.
func TestCancelUsesLifecycleAndRejectsTerminal(t *testing.T) {
	type testCase struct {
		name    string
		state   execution.State
		version int64
		wantErr bool
	}
	tests := []testCase{{name: "created", state: execution.StateCreated, version: 1}, {name: "running", state: execution.StateRunning, version: 2}, {name: "stale", state: execution.StateRunning, version: 1, wantErr: true}, {name: "completed", state: execution.StateCompleted, version: 3, wantErr: true}, {name: "already cancelled", state: execution.StateCancelled, version: 3, wantErr: true}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := setup(t)
			stored := snapshot(t, f.now, tt.state)
			f.tx.On("WithTransaction", mock.Anything, mock.Anything).Return(func(ctx context.Context, fn func(context.Context) error) error { return fn(ctx) })
			f.store.On("Update", mock.Anything, executionID, mock.Anything).Return(func(_ context.Context, _ string, fn func(*execution.Snapshot) (*execution.TransitionFact, error)) error {
				_, err := fn(&stored)
				return err
			})
			if !tt.wantErr {
				f.commands.On("FindOrInsert", mock.Anything, executionID, "cancel", mock.Anything, mock.Anything, executionID).Return(executionID, nil)
			}
			got, err := f.adapter.Cancel(context.Background(), service.CancelRequest{ExecutionID: executionID, ExpectedVersion: tt.version, Reason: service.CancelReasonCandidateEarlyStop})
			if tt.wantErr {
				require.ErrorIs(t, err, service.ErrInterviewConflict)
				return
			}
			require.NoError(t, err)
			require.Equal(t, "CANCELLED", got.State)
			require.Equal(t, service.CancelReasonCandidateEarlyStop, stored.TerminalReason)
		})
	}
}

// TestStartIntentConflictUsesOneCommandIdentity proves both key and consent affect the stored candidate intent.
func TestStartIntentConflictUsesOneCommandIdentity(t *testing.T) {
	f := setup(t)
	f.assignments.On("WithAssignment", mock.Anything, assignmentID, mock.Anything).Return(func(ctx context.Context, _ string, fn func(context.Context, executionport.AssignmentRow) error) error {
		return fn(ctx, executionport.AssignmentRow{InstanceID: "instance", RevisionID: "revision"})
	})
	f.store.On("FindByAssignment", mock.Anything, assignmentID).Return(snapshot(t, f.now, execution.StateCreated), nil)
	f.commands.On("FindOrInsert", mock.Anything, assignmentID, "start", "candidate-start", mock.Anything, executionID).Return("", executionport.ErrIdempotencyConflict)
	_, err := f.adapter.CommitOrFind(context.Background(), service.CommitOrFindRequest{AssignmentID: assignmentID, ConsentID: consentID, IdempotencyKey: "key"})
	require.ErrorIs(t, err, service.ErrInterviewConflict)
}
