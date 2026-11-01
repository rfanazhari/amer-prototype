package executionusecase_test

import (
	"context"
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

const (
	// progressRef is the committed progress token used by the lifecycle tests.
	progressRef = "progress-001"
	// opKey is the opaque operation key used by the lifecycle tests.
	opKey = "op-1"
	// runtimeActor is the actor reference of the interview runtime in the lifecycle tests.
	runtimeActor = "interview-runtime"
	// failReason is the terminal reason token used by the Fail tests.
	failReason = "provider_lost"
)

// lifecycleFixture wires the lifecycle operations to strict mocks and captures what Update committed.
type lifecycleFixture struct {
	tx       *mocks.Transactor
	store    *mocks.ExecutionStore
	commands *mocks.CommandStore
	progress *mocks.ProgressCommitment
	// stored is the locked row handed to the Update callback.
	stored execution.Snapshot
	// after is the snapshot the callback left behind when it succeeded.
	after execution.Snapshot
	// fact is the transition fact the callback returned, nil when none was recorded.
	fact *execution.TransitionFact
	// updateErr, when set, is returned by Update without running the callback.
	updateErr error
}

// newLifecycleFixture builds strict mocks whose transaction and Update run the callback against stored.
func newLifecycleFixture(t *testing.T, stored execution.Snapshot, wrap func(error) error) *lifecycleFixture {
	t.Helper()
	f := &lifecycleFixture{
		tx: mocks.NewTransactor(t), store: mocks.NewExecutionStore(t),
		commands: mocks.NewCommandStore(t), progress: mocks.NewProgressCommitment(t),
		stored: stored, after: stored,
	}
	f.tx.On("WithTransaction", mock.Anything, mock.Anything).Return(
		func(ctx context.Context, fn func(context.Context) error) error {
			err := fn(ctx)
			if wrap != nil {
				return wrap(err)
			}
			return err
		}).Maybe()
	f.store.On("Update", mock.Anything, existingID, mock.Anything).Return(
		func(_ context.Context, _ string, fn func(*execution.Snapshot) (*execution.TransitionFact, error)) error {
			if f.updateErr != nil {
				return f.updateErr
			}
			snap := f.stored
			fact, err := fn(&snap)
			if err == nil {
				f.after, f.fact = snap, fact
			}
			return err
		}).Maybe()
	return f
}

// expectCommand expects one idempotent command row for the execution and returns the execution id.
func (f *lifecycleFixture) expectCommand(kind, key string) {
	f.commands.On("FindOrInsert", mock.Anything, existingID, kind, key, mock.Anything, existingID).Return(existingID, nil).Once()
}

// requireNothingWritten asserts the callback left the stored row exactly as it found it.
func (f *lifecycleFixture) requireNothingWritten(t *testing.T) {
	t.Helper()
	require.Equal(t, f.stored, f.after)
	require.Nil(t, f.fact)
}

// activateOp builds Activate over the fixture ports.
func (f *lifecycleFixture) activateOp(t *testing.T) executionusecase.Activate {
	return executionusecase.Activate{Transactions: f.tx, Executions: f.store, Commands: f.commands, Clock: testClock(t)}
}

// progressOp builds CommitProgress over the fixture ports.
func (f *lifecycleFixture) progressOp(t *testing.T) executionusecase.CommitProgress {
	return executionusecase.CommitProgress{Transactions: f.tx, Executions: f.store, Clock: testClock(t)}
}

// completeOp builds Complete over the fixture ports.
func (f *lifecycleFixture) completeOp(t *testing.T) executionusecase.Complete {
	return executionusecase.Complete{Transactions: f.tx, Executions: f.store, Commands: f.commands, Progress: f.progress, Clock: testClock(t)}
}

// failOp builds Fail over the fixture ports.
func (f *lifecycleFixture) failOp(t *testing.T) executionusecase.Fail {
	return executionusecase.Fail{Transactions: f.tx, Executions: f.store, Commands: f.commands, Clock: testClock(t)}
}

// terminalSnapshot builds a stored terminal row of the requested state for the shared test binding.
func terminalSnapshot(t *testing.T, state execution.State) execution.Snapshot {
	t.Helper()
	if state == execution.StateCancelled {
		return snapshotIn(t, execution.StateCancelled)
	}
	agg, err := execution.RestoreExecution(snapshotIn(t, execution.StateRunning))
	require.NoError(t, err)
	at := trustedNow().Add(-time.Minute)
	var snap execution.Snapshot
	if state == execution.StateCompleted {
		snap, _, err = agg.Complete(progressRef, 2, 1, nil, runtimeActor, opKey, at)
	} else {
		snap, _, err = agg.Fail(failReason, 2, 1, nil, runtimeActor, opKey, at)
	}
	require.NoError(t, err)
	return snap
}

// withGeneration returns snap carrying the given current attempt generation.
func withGeneration(snap execution.Snapshot, generation int64) execution.Snapshot {
	snap.CurrentGeneration = generation
	return snap
}

// timePtr returns a pointer to a copy of value.
func timePtr(value time.Time) *time.Time { return &value }

// activateInput is a well-formed activation of the CREATED test execution.
func activateInput() executionusecase.ActivateInput {
	return executionusecase.ActivateInput{ExecutionID: existingID, RequestKey: "act-001", ActorRef: runtimeActor, ExpectedVersion: 1, OperationKey: opKey, Generation: 1}
}

// progressInput is a well-formed progress commit against the RUNNING test execution.
func progressInput() executionusecase.CommitProgressInput {
	return executionusecase.CommitProgressInput{ExecutionID: existingID, RequestKey: "prog-001", ActorRef: runtimeActor, ExpectedVersion: 2, OperationKey: opKey, Generation: 1, ProgressRef: progressRef}
}

// completeInput is a well-formed completion of the RUNNING test execution.
func completeInput() executionusecase.CompleteInput {
	return executionusecase.CompleteInput{ExecutionID: existingID, RequestKey: "done-001", ActorRef: runtimeActor, ExpectedVersion: 2, OperationKey: opKey, Generation: 1, ProgressRef: progressRef}
}

// failInput is a well-formed failure of the RUNNING test execution.
func failInput() executionusecase.FailInput {
	return executionusecase.FailInput{ExecutionID: existingID, RequestKey: "fail-001", ActorRef: runtimeActor, ExpectedVersion: 2, OperationKey: opKey, Generation: 1, Reason: failReason}
}

// TestActivateMovesTheSameExecutionToRunning covers the success path and its committed facts.
func TestActivateMovesTheSameExecutionToRunning(t *testing.T) {
	f := newLifecycleFixture(t, snapshotIn(t, execution.StateCreated), nil)
	f.expectCommand("activate", "act-001")
	input := activateInput()
	input.Deadline = timePtr(trustedNow().Add(time.Second))

	view, err := f.activateOp(t).Execute(context.Background(), input)

	require.NoError(t, err)
	require.Equal(t, existingID, view.ID)
	require.Equal(t, "RUNNING", view.State)
	require.EqualValues(t, 2, view.StateVersion)
	require.Equal(t, instanceID, view.InstanceID)
	require.Equal(t, revisionID, view.RevisionID)
	require.NotNil(t, view.ActivatedAt)
	require.Equal(t, trustedNow(), *view.ActivatedAt)
	require.Nil(t, view.TerminalAt)
	require.NotNil(t, f.fact)
	require.Equal(t, execution.StateRunning, f.fact.ToState)
	require.Equal(t, runtimeActor, f.fact.ActorRef)
	require.Equal(t, opKey, f.fact.OperationKey)
}

// TestActivateFailureLeavesTheExecutionCreated covers stale fencing, a stale version and an exact deadline.
func TestActivateFailureLeavesTheExecutionCreated(t *testing.T) {
	cases := []struct {
		name   string
		stored execution.Snapshot
		mutate func(*executionusecase.ActivateInput)
		code   string
	}{
		{"stale generation", withGeneration(snapshotIn(t, execution.StateCreated), 3), func(*executionusecase.ActivateInput) {}, errs.EXLStaleAttempt},
		{"deadline equal to now", snapshotIn(t, execution.StateCreated), func(in *executionusecase.ActivateInput) { in.Deadline = timePtr(trustedNow()) }, errs.EXLStaleAttempt},
		{"stale version", snapshotIn(t, execution.StateCreated), func(in *executionusecase.ActivateInput) { in.ExpectedVersion = 9 }, errs.EXLStaleVersion},
		{"already terminal", terminalSnapshot(t, execution.StateCancelled), func(*executionusecase.ActivateInput) {}, errs.EXLInvalidTransition},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := newLifecycleFixture(t, tc.stored, nil)
			input := activateInput()
			tc.mutate(&input)

			view, err := f.activateOp(t).Execute(context.Background(), input)

			require.Nil(t, view)
			requireCode(t, err, tc.code)
			f.requireNothingWritten(t)
		})
	}
}

// TestActivateReplayReturnsTheRunningExecutionWithoutWriting covers a lost-response retry of the same activation.
func TestActivateReplayReturnsTheRunningExecutionWithoutWriting(t *testing.T) {
	f := newLifecycleFixture(t, snapshotIn(t, execution.StateRunning), nil)

	view, err := f.activateOp(t).Execute(context.Background(), activateInput())

	require.NoError(t, err)
	require.Equal(t, "RUNNING", view.State)
	require.EqualValues(t, 2, view.StateVersion)
	f.requireNothingWritten(t)
}

// TestCommitProgressWhileRunningDoesNotComplete covers the stored reference and the unchanged state.
func TestCommitProgressWhileRunningDoesNotComplete(t *testing.T) {
	f := newLifecycleFixture(t, snapshotIn(t, execution.StateRunning), nil)

	view, err := f.progressOp(t).Execute(context.Background(), progressInput())

	require.NoError(t, err)
	require.Equal(t, "RUNNING", view.State)
	require.EqualValues(t, 2, view.StateVersion)
	require.Equal(t, progressRef, view.ProgressRef)
	require.Nil(t, view.TerminalAt)
	require.Nil(t, f.fact)
	require.Equal(t, execution.Binding{InstanceID: instanceID, RevisionID: revisionID}, f.after.Binding)
}

// TestCommitProgressRejectsStaleOrIllegalWrites covers fencing, version, deadline and source-state checks.
func TestCommitProgressRejectsStaleOrIllegalWrites(t *testing.T) {
	cases := []struct {
		name   string
		stored execution.Snapshot
		mutate func(*executionusecase.CommitProgressInput)
		code   string
	}{
		{"stale generation", withGeneration(snapshotIn(t, execution.StateRunning), 4), func(*executionusecase.CommitProgressInput) {}, errs.EXLStaleAttempt},
		{"deadline equal to now", snapshotIn(t, execution.StateRunning), func(in *executionusecase.CommitProgressInput) { in.Deadline = timePtr(trustedNow()) }, errs.EXLStaleAttempt},
		{"stale version", snapshotIn(t, execution.StateRunning), func(in *executionusecase.CommitProgressInput) { in.ExpectedVersion = 7 }, errs.EXLStaleVersion},
		{"created", snapshotIn(t, execution.StateCreated), func(*executionusecase.CommitProgressInput) {}, errs.EXLInvalidTransition},
		{"completed", terminalSnapshot(t, execution.StateCompleted), func(*executionusecase.CommitProgressInput) {}, errs.EXLInvalidTransition},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := newLifecycleFixture(t, tc.stored, nil)
			input := progressInput()
			tc.mutate(&input)

			view, err := f.progressOp(t).Execute(context.Background(), input)

			require.Nil(t, view)
			requireCode(t, err, tc.code)
			f.requireNothingWritten(t)
		})
	}
}

// TestCompleteCommitsOnlyWhenTheProgressPortConfirms covers port true and the stored reference.
func TestCompleteCommitsOnlyWhenTheProgressPortConfirms(t *testing.T) {
	f := newLifecycleFixture(t, snapshotIn(t, execution.StateRunning), nil)
	f.progress.On("IsCommitted", mock.Anything, existingID, execution.ProgressRef(progressRef)).Return(true, nil)
	f.expectCommand("complete", "done-001")

	view, err := f.completeOp(t).Execute(context.Background(), completeInput())

	require.NoError(t, err)
	require.Equal(t, "COMPLETED", view.State)
	require.EqualValues(t, 3, view.StateVersion)
	require.Equal(t, progressRef, view.ProgressRef)
	require.NotNil(t, view.TerminalAt)
	require.Equal(t, trustedNow(), *view.TerminalAt)
	require.Equal(t, instanceID, view.InstanceID)
	require.Equal(t, revisionID, view.RevisionID)
	require.NotNil(t, f.fact)
	require.Equal(t, execution.StateCompleted, f.fact.ToState)
}

// TestCompleteWithoutCommittedProgressWritesNothing covers port false and an unreadable progress store.
func TestCompleteWithoutCommittedProgressWritesNothing(t *testing.T) {
	cases := []struct {
		name      string
		committed bool
		portErr   error
		code      string
	}{
		{"progress not committed", false, nil, errs.EXLProgressRequired},
		{"progress store unreadable", false, executionport.ErrDependencyUnavailable, errs.EXLDependencyUnavailable},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := newLifecycleFixture(t, snapshotIn(t, execution.StateRunning), nil)
			f.progress.On("IsCommitted", mock.Anything, existingID, execution.ProgressRef(progressRef)).Return(tc.committed, tc.portErr)

			view, err := f.completeOp(t).Execute(context.Background(), completeInput())

			require.Nil(t, view)
			requireCode(t, err, tc.code)
			f.requireNothingWritten(t)
			require.Equal(t, execution.StateRunning, f.after.State)
		})
	}
}

// TestCompleteStaleChecksWriteNothingAndSkipThePort covers version, generation, deadline and state checks.
func TestCompleteStaleChecksWriteNothingAndSkipThePort(t *testing.T) {
	cases := []struct {
		name   string
		stored execution.Snapshot
		mutate func(*executionusecase.CompleteInput)
		code   string
	}{
		{"stale version", snapshotIn(t, execution.StateRunning), func(in *executionusecase.CompleteInput) { in.ExpectedVersion = 5 }, errs.EXLStaleVersion},
		{"stale generation", withGeneration(snapshotIn(t, execution.StateRunning), 6), func(*executionusecase.CompleteInput) {}, errs.EXLStaleAttempt},
		{"deadline equal to now", snapshotIn(t, execution.StateRunning), func(in *executionusecase.CompleteInput) { in.Deadline = timePtr(trustedNow()) }, errs.EXLStaleAttempt},
		{"not yet running", snapshotIn(t, execution.StateCreated), func(*executionusecase.CompleteInput) {}, errs.EXLInvalidTransition},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := newLifecycleFixture(t, tc.stored, nil)
			input := completeInput()
			tc.mutate(&input)

			view, err := f.completeOp(t).Execute(context.Background(), input)

			require.Nil(t, view)
			requireCode(t, err, tc.code)
			f.requireNothingWritten(t)
		})
	}
}

// TestCompleteRepeatReturnsTheFirstOutcomeAndOtherTerminalsConflict covers INV-04 and BR-CORE-054.
func TestCompleteRepeatReturnsTheFirstOutcomeAndOtherTerminalsConflict(t *testing.T) {
	cases := []struct {
		name   string
		stored execution.Snapshot
		mutate func(*executionusecase.CompleteInput)
		code   string
	}{
		{"same completion", terminalSnapshot(t, execution.StateCompleted), func(*executionusecase.CompleteInput) {}, ""},
		{"same completion with a stale version", terminalSnapshot(t, execution.StateCompleted), func(in *executionusecase.CompleteInput) { in.ExpectedVersion = 1 }, ""},
		{"different progress", terminalSnapshot(t, execution.StateCompleted), func(in *executionusecase.CompleteInput) { in.ProgressRef = "progress-002" }, errs.EXLTerminalConflict},
		{"after failure", terminalSnapshot(t, execution.StateFailed), func(*executionusecase.CompleteInput) {}, errs.EXLTerminalConflict},
		{"after cancellation", terminalSnapshot(t, execution.StateCancelled), func(*executionusecase.CompleteInput) {}, errs.EXLTerminalConflict},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := newLifecycleFixture(t, tc.stored, nil)
			input := completeInput()
			tc.mutate(&input)

			view, err := f.completeOp(t).Execute(context.Background(), input)

			f.requireNothingWritten(t)
			if tc.code != "" {
				require.Nil(t, view)
				requireCode(t, err, tc.code)
				return
			}
			require.NoError(t, err)
			require.Equal(t, "COMPLETED", view.State)
			require.Equal(t, progressRef, view.ProgressRef)
			require.Equal(t, tc.stored.Binding.RevisionID, view.RevisionID)
			require.Equal(t, tc.stored.TerminalAt, view.TerminalAt)
		})
	}
}

// TestFailIsAllowedOnceFromCreatedAndRunning covers both legal sources and the committed facts.
func TestFailIsAllowedOnceFromCreatedAndRunning(t *testing.T) {
	cases := []struct {
		name    string
		stored  execution.Snapshot
		version int64
	}{
		{"created", snapshotIn(t, execution.StateCreated), 1},
		{"running", snapshotIn(t, execution.StateRunning), 2},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := newLifecycleFixture(t, tc.stored, nil)
			f.expectCommand("fail", "fail-001")
			input := failInput()
			input.ExpectedVersion = tc.version

			view, err := f.failOp(t).Execute(context.Background(), input)

			require.NoError(t, err)
			require.Equal(t, "FAILED", view.State)
			require.Equal(t, failReason, view.TerminalReason)
			require.EqualValues(t, tc.version+1, view.StateVersion)
			require.NotNil(t, view.TerminalAt)
			require.NotNil(t, f.fact)
			require.Equal(t, execution.StateFailed, f.fact.ToState)
			require.Equal(t, failReason, f.fact.Reason)
		})
	}
}

// TestFailRepeatReturnsTheFirstOutcomeAndOtherTerminalsConflict covers INV-04 and BR-CORE-054 for Fail.
func TestFailRepeatReturnsTheFirstOutcomeAndOtherTerminalsConflict(t *testing.T) {
	cases := []struct {
		name   string
		stored execution.Snapshot
		mutate func(*executionusecase.FailInput)
		code   string
	}{
		{"same failure", terminalSnapshot(t, execution.StateFailed), func(*executionusecase.FailInput) {}, ""},
		{"different reason", terminalSnapshot(t, execution.StateFailed), func(in *executionusecase.FailInput) { in.Reason = "other_reason" }, errs.EXLTerminalConflict},
		{"after completion", terminalSnapshot(t, execution.StateCompleted), func(*executionusecase.FailInput) {}, errs.EXLTerminalConflict},
		{"after cancellation", terminalSnapshot(t, execution.StateCancelled), func(*executionusecase.FailInput) {}, errs.EXLTerminalConflict},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := newLifecycleFixture(t, tc.stored, nil)
			input := failInput()
			tc.mutate(&input)

			view, err := f.failOp(t).Execute(context.Background(), input)

			f.requireNothingWritten(t)
			if tc.code != "" {
				require.Nil(t, view)
				requireCode(t, err, tc.code)
				return
			}
			require.NoError(t, err)
			require.Equal(t, "FAILED", view.State)
			require.Equal(t, failReason, view.TerminalReason)
		})
	}
}

// TestFailStaleChecksWriteNothing covers version, generation and deadline fencing for Fail.
func TestFailStaleChecksWriteNothing(t *testing.T) {
	cases := []struct {
		name   string
		stored execution.Snapshot
		mutate func(*executionusecase.FailInput)
		code   string
	}{
		{"stale version", snapshotIn(t, execution.StateRunning), func(in *executionusecase.FailInput) { in.ExpectedVersion = 8 }, errs.EXLStaleVersion},
		{"stale generation", withGeneration(snapshotIn(t, execution.StateRunning), 5), func(*executionusecase.FailInput) {}, errs.EXLStaleAttempt},
		{"deadline equal to now", snapshotIn(t, execution.StateRunning), func(in *executionusecase.FailInput) { in.Deadline = timePtr(trustedNow()) }, errs.EXLStaleAttempt},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := newLifecycleFixture(t, tc.stored, nil)
			input := failInput()
			tc.mutate(&input)

			view, err := f.failOp(t).Execute(context.Background(), input)

			require.Nil(t, view)
			requireCode(t, err, tc.code)
			f.requireNothingWritten(t)
		})
	}
}

// TestLifecycleCommandsRejectBadShapeBeforeAnyTransaction covers malformed identifiers, tokens and numbers.
func TestLifecycleCommandsRejectBadShapeBeforeAnyTransaction(t *testing.T) {
	f := newLifecycleFixture(t, snapshotIn(t, execution.StateRunning), nil)
	f.tx.ExpectedCalls = nil
	ctx := context.Background()

	activate := activateInput()
	activate.ExecutionID = "not-a-uuid"
	_, err := f.activateOp(t).Execute(ctx, activate)
	requireCode(t, err, errs.InvalidInput)

	activate = activateInput()
	activate.Generation = 0
	_, err = f.activateOp(t).Execute(ctx, activate)
	requireCode(t, err, errs.InvalidInput)

	progress := progressInput()
	progress.ProgressRef = "has space"
	_, err = f.progressOp(t).Execute(ctx, progress)
	requireCode(t, err, errs.InvalidInput)

	complete := completeInput()
	complete.ProgressRef = ""
	_, err = f.completeOp(t).Execute(ctx, complete)
	requireCode(t, err, errs.InvalidInput)

	complete = completeInput()
	complete.ExpectedVersion = 0
	_, err = f.completeOp(t).Execute(ctx, complete)
	requireCode(t, err, errs.InvalidInput)

	fail := failInput()
	fail.Reason = ""
	_, err = f.failOp(t).Execute(ctx, fail)
	requireCode(t, err, errs.InvalidInput)

	fail = failInput()
	fail.RequestKey = "bad key"
	_, err = f.failOp(t).Execute(ctx, fail)
	requireCode(t, err, errs.InvalidInput)
}

// TestLifecycleCommandsMapStoreFailures covers missing rows, command conflicts and an ambiguous commit.
func TestLifecycleCommandsMapStoreFailures(t *testing.T) {
	t.Run("execution not found", func(t *testing.T) {
		f := newLifecycleFixture(t, snapshotIn(t, execution.StateCreated), nil)
		f.updateErr = executionport.ErrNotFound

		_, err := f.activateOp(t).Execute(context.Background(), activateInput())

		requireCode(t, err, errs.EXLNotFound)
	})
	t.Run("same key with a different intent", func(t *testing.T) {
		f := newLifecycleFixture(t, snapshotIn(t, execution.StateRunning), nil)
		f.progress.On("IsCommitted", mock.Anything, existingID, execution.ProgressRef(progressRef)).Return(true, nil)
		f.commands.On("FindOrInsert", mock.Anything, existingID, "complete", "done-001", mock.Anything, existingID).Return("", executionport.ErrIdempotencyConflict)

		_, err := f.completeOp(t).Execute(context.Background(), completeInput())

		requireCode(t, err, errs.EXLIdempotencyConflict)
	})
	t.Run("ambiguous commit", func(t *testing.T) {
		f := newLifecycleFixture(t, snapshotIn(t, execution.StateCreated), func(error) error { return executionport.ErrUncertainCommit })
		f.expectCommand("activate", "act-001")

		view, err := f.activateOp(t).Execute(context.Background(), activateInput())

		require.Nil(t, view)
		requireCode(t, err, errs.EXLUncertainCommit)
	})
	t.Run("unexpected failure is technical", func(t *testing.T) {
		f := newLifecycleFixture(t, snapshotIn(t, execution.StateCreated), nil)
		f.updateErr = context.DeadlineExceeded

		_, err := f.failOp(t).Execute(context.Background(), failInput())

		requireTechnical(t, err)
	})
}

// compile-time check that the lifecycle operations satisfy their operation interfaces.
var (
	_ executionusecase.ActivateOperation       = executionusecase.Activate{}
	_ executionusecase.CommitProgressOperation = executionusecase.CommitProgress{}
	_ executionusecase.CompleteOperation       = executionusecase.Complete{}
	_ executionusecase.FailOperation           = executionusecase.Fail{}
)
