package executionusecase_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/rfanazhari/amer-prototype/application/repository/executionport"
	"github.com/rfanazhari/amer-prototype/application/usecase/executionusecase"
	"github.com/rfanazhari/amer-prototype/domain/aggregate/execution"
	"github.com/rfanazhari/amer-prototype/pkg/errs"
	"github.com/rfanazhari/amer-prototype/test/mocks"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// startFixture wires Start to strict mocks; any call without an expectation fails the test.
type startFixture struct {
	lock     *mocks.AssignmentLock
	store    *mocks.ExecutionStore
	commands *mocks.CommandStore
	start    executionusecase.Start
}

// newStartFixture builds a Start whose minted id is newID.
func newStartFixture(t *testing.T) startFixture {
	t.Helper()
	f := startFixture{lock: mocks.NewAssignmentLock(t), store: mocks.NewExecutionStore(t), commands: mocks.NewCommandStore(t)}
	f.start = executionusecase.Start{Assignments: f.lock, Executions: f.store, Commands: f.commands, Clock: testClock(t), NewID: fixedID(newID)}
	return f
}

// validStart is a well-formed start command.
func validStart() executionusecase.StartInput {
	return executionusecase.StartInput{AssignmentID: assignmentID, RequestKey: requestKey, ActorRef: actorRef, InputRefs: []string{"doc-b", "doc-a"}}
}

// TestStartCommitsOneCreatedExecutionBoundToAssignmentRevision covers a first start: CREATED, no provider call, revision from the assignment.
func TestStartCommitsOneCreatedExecutionBoundToAssignmentRevision(t *testing.T) {
	f := newStartFixture(t)
	lockRunsWith(f.lock, openRow(), nil)
	f.store.On("FindByAssignment", mock.Anything, assignmentID).Return(execution.Snapshot{}, executionport.ErrNotFound)
	var (
		gotSnap execution.Snapshot
		gotFact execution.TransitionFact
		gotPin  string
		gotKind string
		gotScp  string
		gotKey  string
		gotFP   string
		gotExec string
	)
	f.store.On("InsertInterview", mock.Anything, assignmentID, mock.Anything, mock.Anything).Return(
		func(_ context.Context, pin string, snap execution.Snapshot, fact execution.TransitionFact) error {
			gotPin, gotSnap, gotFact = pin, snap, fact
			return nil
		})
	f.commands.On("FindOrInsert", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(
		func(_ context.Context, scope, kind, key, fingerprint, executionID string) (string, error) {
			gotScp, gotKind, gotKey, gotFP, gotExec = scope, kind, key, fingerprint, executionID
			return executionID, nil
		})

	view, err := f.start.Execute(context.Background(), validStart())

	require.NoError(t, err)
	require.NotNil(t, view)
	require.Equal(t, newID, view.ID)
	require.Equal(t, assignmentID, view.AssignmentID)
	require.Equal(t, instanceID, view.InstanceID)
	require.Equal(t, revisionID, view.RevisionID)
	require.Equal(t, "CREATED", view.State)
	require.EqualValues(t, 1, view.StateVersion)
	require.Nil(t, view.ActivatedAt)
	require.Nil(t, view.TerminalAt)
	require.Empty(t, view.ProgressRef)
	require.Equal(t, trustedNow(), view.CreatedAt)
	require.Equal(t, assignmentID, gotPin)
	require.Equal(t, execution.StateCreated, gotSnap.State)
	require.Equal(t, execution.Binding{InstanceID: instanceID, RevisionID: revisionID}, gotSnap.Binding)
	require.Equal(t, trustedNow(), gotSnap.CreatedAt)
	require.Equal(t, execution.StateCreated, gotFact.ToState)
	require.Equal(t, assignmentID, gotScp)
	require.Equal(t, "start", gotKind)
	require.Equal(t, requestKey, gotKey)
	require.Equal(t, newID, gotExec)
	require.Regexp(t, `^[0-9a-f]{64}$`, gotFP)
}

// TestStartFingerprintIgnoresRefOrderAndTracksBinding proves the intent hash is order-independent and binding-sensitive.
func TestStartFingerprintIgnoresRefOrderAndTracksBinding(t *testing.T) {
	run := func(t *testing.T, row executionport.AssignmentRow, refs []string) string {
		f := newStartFixture(t)
		lockRunsWith(f.lock, row, nil)
		f.store.On("FindByAssignment", mock.Anything, assignmentID).Return(execution.Snapshot{}, executionport.ErrNotFound)
		f.store.On("InsertInterview", mock.Anything, assignmentID, mock.Anything, mock.Anything).Return(nil)
		var fp string
		f.commands.On("FindOrInsert", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(
			func(_ context.Context, _, _, _, fingerprint, id string) (string, error) {
				fp = fingerprint
				return id, nil
			})
		in := validStart()
		in.InputRefs = refs
		_, err := f.start.Execute(context.Background(), in)
		require.NoError(t, err)
		return fp
	}
	base := run(t, openRow(), []string{"doc-b", "doc-a"})
	require.Equal(t, base, run(t, openRow(), []string{"doc-a", "doc-b", "doc-a"}))
	moved := openRow()
	moved.RevisionID = otherRevID
	require.NotEqual(t, base, run(t, moved, []string{"doc-b", "doc-a"}))
	require.NotEqual(t, base, run(t, openRow(), []string{"doc-a"}))
}

// TestStartReplayReturnsOriginalExecution covers the same key and a second key for one assignment.
func TestStartReplayReturnsOriginalExecution(t *testing.T) {
	for _, key := range []string{requestKey, "start-002"} {
		t.Run(key, func(t *testing.T) {
			f := newStartFixture(t)
			lockRunsWith(f.lock, openRow(), nil)
			f.store.On("FindByAssignment", mock.Anything, assignmentID).Return(snapshotIn(t, execution.StateRunning), nil)
			f.commands.On("FindOrInsert", mock.Anything, assignmentID, "start", key, mock.Anything, existingID).Return(existingID, nil)
			in := validStart()
			in.RequestKey = key

			view, err := f.start.Execute(context.Background(), in)

			require.NoError(t, err)
			require.NotNil(t, view)
			require.Equal(t, existingID, view.ID)
			require.Equal(t, "RUNNING", view.State)
			require.Equal(t, assignmentID, view.AssignmentID)
		})
	}
}

// TestStartConflictsWriteNothing covers a reused key with a different intent and a pin bound to something else.
func TestStartConflictsWriteNothing(t *testing.T) {
	t.Run("same key different fingerprint", func(t *testing.T) {
		f := newStartFixture(t)
		lockRunsWith(f.lock, openRow(), nil)
		f.store.On("FindByAssignment", mock.Anything, assignmentID).Return(snapshotIn(t, execution.StateCreated), nil)
		f.commands.On("FindOrInsert", mock.Anything, assignmentID, "start", requestKey, mock.Anything, existingID).Return("", executionport.ErrIdempotencyConflict)

		view, err := f.start.Execute(context.Background(), validStart())

		require.Nil(t, view)
		requireCode(t, err, errs.EXLIdempotencyConflict)
	})
	t.Run("pin binds a different revision", func(t *testing.T) {
		f := newStartFixture(t)
		row := openRow()
		row.RevisionID = otherRevID
		lockRunsWith(f.lock, row, nil)
		f.store.On("FindByAssignment", mock.Anything, assignmentID).Return(snapshotIn(t, execution.StateCreated), nil)

		view, err := f.start.Execute(context.Background(), validStart())

		require.Nil(t, view)
		requireCode(t, err, errs.EXLIdempotencyConflict)
	})
}

// TestStartUnavailableAssignment covers revoked and completed assignments, including cancellation in the same transaction.
func TestStartUnavailableAssignment(t *testing.T) {
	revoked := openRow()
	at := trustedNow().Add(-time.Minute)
	revoked.AccessRevokedAt = &at
	completed := openRow()
	completed.CompletedAt = &at

	t.Run("revoked without execution inserts nothing", func(t *testing.T) {
		f := newStartFixture(t)
		lockRunsWith(f.lock, revoked, nil)
		f.store.On("FindByAssignment", mock.Anything, assignmentID).Return(execution.Snapshot{}, executionport.ErrNotFound)

		view, err := f.start.Execute(context.Background(), validStart())

		require.Nil(t, view)
		requireCode(t, err, errs.EXLAssignmentUnavailable)
	})
	t.Run("completed without execution inserts nothing", func(t *testing.T) {
		f := newStartFixture(t)
		lockRunsWith(f.lock, completed, nil)
		f.store.On("FindByAssignment", mock.Anything, assignmentID).Return(execution.Snapshot{}, executionport.ErrNotFound)

		view, err := f.start.Execute(context.Background(), validStart())

		require.Nil(t, view)
		requireCode(t, err, errs.EXLAssignmentUnavailable)
	})
	for _, state := range []execution.State{execution.StateCreated, execution.StateRunning} {
		t.Run("revoked cancels "+string(state)+" and still reports unavailable", func(t *testing.T) {
			f := newStartFixture(t)
			lockRunsWith(f.lock, revoked, nil)
			f.store.On("FindByAssignment", mock.Anything, assignmentID).Return(snapshotIn(t, state), nil)
			var after execution.Snapshot
			var fact *execution.TransitionFact
			f.store.On("Update", mock.Anything, existingID, mock.Anything).Return(
				func(_ context.Context, _ string, fn func(*execution.Snapshot) (*execution.TransitionFact, error)) error {
					snap := snapshotIn(t, state)
					var err error
					fact, err = fn(&snap)
					after = snap
					return err
				})

			view, err := f.start.Execute(context.Background(), validStart())

			require.Nil(t, view)
			requireCode(t, err, errs.EXLAssignmentUnavailable)
			require.Equal(t, execution.StateCancelled, after.State)
			require.Equal(t, "access_revoked", after.TerminalReason)
			require.NotNil(t, fact)
			require.Equal(t, execution.StateCancelled, fact.ToState)
		})
	}
	t.Run("revoked terminal execution is left as is", func(t *testing.T) {
		f := newStartFixture(t)
		lockRunsWith(f.lock, revoked, nil)
		f.store.On("FindByAssignment", mock.Anything, assignmentID).Return(snapshotIn(t, execution.StateCancelled), nil)

		view, err := f.start.Execute(context.Background(), validStart())

		require.Nil(t, view)
		requireCode(t, err, errs.EXLAssignmentUnavailable)
	})
}

// TestStartMissingAssignmentIsNotFound maps the port's absence to EXL_NOT_FOUND.
func TestStartMissingAssignmentIsNotFound(t *testing.T) {
	f := newStartFixture(t)
	f.lock.On("WithAssignment", mock.Anything, assignmentID, mock.Anything).Return(executionport.ErrNotFound)

	view, err := f.start.Execute(context.Background(), validStart())

	require.Nil(t, view)
	requireCode(t, err, errs.EXLNotFound)
}

// TestStartRejectsMalformedInputBeforeLocking proves shape errors take no lock and write nothing.
func TestStartRejectsMalformedInputBeforeLocking(t *testing.T) {
	refs17 := make([]string, 17)
	for i := range refs17 {
		refs17[i] = "ref-" + strings.Repeat("x", i+1)
	}
	tests := []struct {
		name   string
		mutate func(*executionusecase.StartInput)
	}{
		{"empty key", func(in *executionusecase.StartInput) { in.RequestKey = "" }},
		{"key with space", func(in *executionusecase.StartInput) { in.RequestKey = "bad key" }},
		{"key too long", func(in *executionusecase.StartInput) { in.RequestKey = strings.Repeat("k", 65) }},
		{"padded key is not trimmed", func(in *executionusecase.StartInput) { in.RequestKey = " start-001" }},
		{"bad actor", func(in *executionusecase.StartInput) { in.ActorRef = "actor/ref" }},
		{"empty actor", func(in *executionusecase.StartInput) { in.ActorRef = "" }},
		{"assignment not a uuid", func(in *executionusecase.StartInput) { in.AssignmentID = "not-a-uuid" }},
		{"bad input ref", func(in *executionusecase.StartInput) { in.InputRefs = []string{"ok", "no good"} }},
		{"too many input refs", func(in *executionusecase.StartInput) { in.InputRefs = refs17 }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newStartFixture(t)
			in := validStart()
			tt.mutate(&in)

			view, err := f.start.Execute(context.Background(), in)

			require.Nil(t, view)
			requireCode(t, err, errs.InvalidInput)
		})
	}
}

// TestStartCommitFaults covers uncertain commit, missing dependency, technical failures and the pin race.
func TestStartCommitFaults(t *testing.T) {
	t.Run("uncertain commit returns no id", func(t *testing.T) {
		f := newStartFixture(t)
		lockRunsWith(f.lock, openRow(), func(err error) error {
			require.NoError(t, err)
			return executionport.ErrUncertainCommit
		})
		f.store.On("FindByAssignment", mock.Anything, assignmentID).Return(execution.Snapshot{}, executionport.ErrNotFound)
		f.store.On("InsertInterview", mock.Anything, assignmentID, mock.Anything, mock.Anything).Return(nil)
		f.commands.On("FindOrInsert", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(newID, nil)

		view, err := f.start.Execute(context.Background(), validStart())

		require.Nil(t, view)
		requireCode(t, err, errs.EXLUncertainCommit)
	})
	t.Run("missing revision is dependency unavailable", func(t *testing.T) {
		f := newStartFixture(t)
		lockRunsWith(f.lock, openRow(), nil)
		f.store.On("FindByAssignment", mock.Anything, assignmentID).Return(execution.Snapshot{}, executionport.ErrNotFound)
		f.store.On("InsertInterview", mock.Anything, assignmentID, mock.Anything, mock.Anything).Return(executionport.ErrDependencyUnavailable)

		view, err := f.start.Execute(context.Background(), validStart())

		require.Nil(t, view)
		requireCode(t, err, errs.EXLDependencyUnavailable)
	})
	t.Run("unexpected database error stays technical", func(t *testing.T) {
		f := newStartFixture(t)
		lockRunsWith(f.lock, openRow(), nil)
		f.store.On("FindByAssignment", mock.Anything, assignmentID).Return(execution.Snapshot{}, errors.New("connection reset"))

		view, err := f.start.Execute(context.Background(), validStart())

		require.Nil(t, view)
		requireTechnical(t, err)
	})
	t.Run("pin race reconciles by reading the winner", func(t *testing.T) {
		f := newStartFixture(t)
		lockRunsWith(f.lock, openRow(), nil)
		f.store.On("FindByAssignment", mock.Anything, assignmentID).Return(execution.Snapshot{}, executionport.ErrNotFound).Once()
		f.store.On("InsertInterview", mock.Anything, assignmentID, mock.Anything, mock.Anything).Return(executionport.ErrAlreadyExists).Once()
		f.store.On("FindByAssignment", mock.Anything, assignmentID).Return(snapshotIn(t, execution.StateCreated), nil).Once()
		f.commands.On("FindOrInsert", mock.Anything, assignmentID, "start", requestKey, mock.Anything, existingID).Return(existingID, nil)

		view, err := f.start.Execute(context.Background(), validStart())

		require.NoError(t, err)
		require.NotNil(t, view)
		require.Equal(t, existingID, view.ID)
	})
}
