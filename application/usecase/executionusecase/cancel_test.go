package executionusecase_test

import (
	"context"
	"errors"
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
	// stopReason is the terminal reason token used by the early-stop Cancel tests.
	stopReason = "candidate_stopped"
	// stopActor is the actor reference of Candidate Interview in the Cancel tests.
	stopActor = "candidate-interview"
)

// cancelOp builds Cancel over the fixture ports.
func (f *lifecycleFixture) cancelOp(t *testing.T) executionusecase.Cancel {
	return executionusecase.Cancel{Transactions: f.tx, Executions: f.store, Commands: f.commands, Clock: testClock(t)}
}

// cancelInput is a well-formed early stop of the CREATED test execution.
func cancelInput() executionusecase.CancelInput {
	return executionusecase.CancelInput{ExecutionID: existingID, RequestKey: "cancel-001", ActorRef: stopActor, ExpectedVersion: 1, Reason: stopReason}
}

// TestCancelMovesNonterminalExecutionToCancelled covers both legal sources and the committed facts.
func TestCancelMovesNonterminalExecutionToCancelled(t *testing.T) {
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
			f.expectCommand("cancel", "cancel-001")
			input := cancelInput()
			input.ExpectedVersion = tc.version

			view, err := f.cancelOp(t).Execute(context.Background(), input)

			require.NoError(t, err)
			require.Equal(t, "CANCELLED", view.State)
			require.Equal(t, stopReason, view.TerminalReason)
			require.EqualValues(t, tc.version+1, view.StateVersion)
			require.NotNil(t, view.TerminalAt)
			require.Equal(t, existingID, view.ID)
			require.Equal(t, tc.stored.Binding, f.after.Binding)
			require.NotNil(t, f.fact)
			require.Equal(t, execution.StateCancelled, f.fact.ToState)
			require.Equal(t, stopReason, f.fact.Reason)
			require.Equal(t, stopActor, f.fact.ActorRef)
		})
	}
}

// TestCancelRepeatReturnsTheFirstOutcomeAndOtherTerminalsConflict covers a second identical cancel and BR-CORE-054.
func TestCancelRepeatReturnsTheFirstOutcomeAndOtherTerminalsConflict(t *testing.T) {
	cancelled := terminalSnapshot(t, execution.StateCancelled)
	cases := []struct {
		name   string
		stored execution.Snapshot
		mutate func(*executionusecase.CancelInput)
		code   string
	}{
		{"same cancel", cancelled, func(in *executionusecase.CancelInput) { in.Reason = cancelled.TerminalReason }, ""},
		{"same cancel with a newer request key", cancelled, func(in *executionusecase.CancelInput) {
			in.Reason, in.RequestKey = cancelled.TerminalReason, "cancel-002"
		}, ""},
		{"different reason", cancelled, func(*executionusecase.CancelInput) {}, errs.EXLTerminalConflict},
		{"after completion", terminalSnapshot(t, execution.StateCompleted), func(*executionusecase.CancelInput) {}, errs.EXLTerminalConflict},
		{"after failure", terminalSnapshot(t, execution.StateFailed), func(*executionusecase.CancelInput) {}, errs.EXLTerminalConflict},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := newLifecycleFixture(t, tc.stored, nil)
			input := cancelInput()
			tc.mutate(&input)

			view, err := f.cancelOp(t).Execute(context.Background(), input)

			f.requireNothingWritten(t)
			if tc.code != "" {
				require.Nil(t, view)
				requireCode(t, err, tc.code)
				return
			}
			require.NoError(t, err)
			require.Equal(t, "CANCELLED", view.State)
			require.Equal(t, cancelled.TerminalReason, view.TerminalReason)
			require.Equal(t, cancelled.StateVersion, view.StateVersion)
		})
	}
}

// TestCancelStaleVersionWritesNothing proves an out-of-date caller cannot cancel a row that moved on.
func TestCancelStaleVersionWritesNothing(t *testing.T) {
	f := newLifecycleFixture(t, snapshotIn(t, execution.StateRunning), nil)
	input := cancelInput()
	input.ExpectedVersion = 1

	view, err := f.cancelOp(t).Execute(context.Background(), input)

	require.Nil(t, view)
	requireCode(t, err, errs.EXLStaleVersion)
	f.requireNothingWritten(t)
}

// TestCancelRejectsBadShapeBeforeAnyTransaction covers malformed identifiers, tokens and versions.
func TestCancelRejectsBadShapeBeforeAnyTransaction(t *testing.T) {
	mutations := map[string]func(*executionusecase.CancelInput){
		"execution id":    func(in *executionusecase.CancelInput) { in.ExecutionID = "not-a-uuid" },
		"request key":     func(in *executionusecase.CancelInput) { in.RequestKey = "" },
		"actor":           func(in *executionusecase.CancelInput) { in.ActorRef = "has space" },
		"reason":          func(in *executionusecase.CancelInput) { in.Reason = "" },
		"expected number": func(in *executionusecase.CancelInput) { in.ExpectedVersion = 0 },
	}
	for name, mutate := range mutations {
		t.Run(name, func(t *testing.T) {
			f := newLifecycleFixture(t, snapshotIn(t, execution.StateCreated), nil)
			f.tx.ExpectedCalls = nil
			input := cancelInput()
			mutate(&input)

			view, err := f.cancelOp(t).Execute(context.Background(), input)

			require.Nil(t, view)
			requireCode(t, err, errs.InvalidInput)
		})
	}
}

// TestCancelMapsStoreFailures keeps absence and ambiguity public and everything else technical.
func TestCancelMapsStoreFailures(t *testing.T) {
	t.Run("missing execution", func(t *testing.T) {
		f := newLifecycleFixture(t, snapshotIn(t, execution.StateCreated), nil)
		f.updateErr = executionport.ErrNotFound

		view, err := f.cancelOp(t).Execute(context.Background(), cancelInput())

		require.Nil(t, view)
		requireCode(t, err, errs.EXLNotFound)
	})
	t.Run("uncertain commit", func(t *testing.T) {
		f := newLifecycleFixture(t, snapshotIn(t, execution.StateCreated), func(err error) error {
			require.NoError(t, err)
			return executionport.ErrUncertainCommit
		})
		f.expectCommand("cancel", "cancel-001")

		view, err := f.cancelOp(t).Execute(context.Background(), cancelInput())

		require.Nil(t, view)
		requireCode(t, err, errs.EXLUncertainCommit)
	})
	t.Run("unexpected failure", func(t *testing.T) {
		f := newLifecycleFixture(t, snapshotIn(t, execution.StateCreated), nil)
		f.updateErr = errors.New("connection reset")

		view, err := f.cancelOp(t).Execute(context.Background(), cancelInput())

		require.Nil(t, view)
		requireTechnical(t, err)
	})
}

// cancelNonterminalFixture wires CancelNonterminal to strict mocks and captures what Update committed.
type cancelNonterminalFixture struct {
	lock  *mocks.AssignmentLock
	store *mocks.ExecutionStore
	op    executionusecase.CancelNonterminal
	// after is the snapshot the Update callback left behind.
	after execution.Snapshot
	// fact is the transition fact the Update callback returned.
	fact *execution.TransitionFact
	// updated reports whether the Update callback ran.
	updated bool
}

// newCancelNonterminalFixture builds an operation whose assignment lock reports the open assignment row.
func newCancelNonterminalFixture(t *testing.T) *cancelNonterminalFixture {
	t.Helper()
	f := &cancelNonterminalFixture{lock: mocks.NewAssignmentLock(t), store: mocks.NewExecutionStore(t)}
	f.op = executionusecase.CancelNonterminal{Assignments: f.lock, Executions: f.store, Clock: testClock(t)}
	return f
}

// expectUpdate expects one Update that hands the callback the locked row.
func (f *cancelNonterminalFixture) expectUpdate(locked execution.Snapshot) {
	f.store.On("Update", mock.Anything, existingID, mock.Anything).Return(
		func(_ context.Context, _ string, fn func(*execution.Snapshot) (*execution.TransitionFact, error)) error {
			snap := locked
			fact, err := fn(&snap)
			f.updated, f.after, f.fact = true, snap, fact
			return err
		}).Once()
}

// accessEnded is a well-formed access-ended cancellation.
func accessEnded(reason string) executionusecase.CancelNonterminalInput {
	return executionusecase.CancelNonterminalInput{AssignmentID: assignmentID, ActorRef: "identity-access", Reason: reason}
}

// TestCancelNonterminalCancelsCreatedAndRunning covers both ended-access reasons under the assignment lock.
func TestCancelNonterminalCancelsCreatedAndRunning(t *testing.T) {
	for _, reason := range []string{"access_revoked", "authority_expired"} {
		for _, state := range []execution.State{execution.StateCreated, execution.StateRunning} {
			t.Run(reason+" "+string(state), func(t *testing.T) {
				f := newCancelNonterminalFixture(t)
				lockRunsWith(f.lock, openRow(), nil)
				f.store.On("FindByAssignment", mock.Anything, assignmentID).Return(snapshotIn(t, state), nil)
				f.expectUpdate(snapshotIn(t, state))

				err := f.op.Execute(context.Background(), accessEnded(reason))

				require.NoError(t, err)
				require.Equal(t, execution.StateCancelled, f.after.State)
				require.Equal(t, reason, f.after.TerminalReason)
				require.NotNil(t, f.after.TerminalAt)
				require.Equal(t, trustedNow(), *f.after.TerminalAt)
				require.Equal(t, snapshotIn(t, state).Binding, f.after.Binding)
				require.NotNil(t, f.fact)
				require.Equal(t, execution.StateCancelled, f.fact.ToState)
				require.Equal(t, "identity-access", f.fact.ActorRef)
			})
		}
	}
}

// TestCancelNonterminalLeavesTerminalExecutionsUnchanged proves a finished run is never written or reopened.
func TestCancelNonterminalLeavesTerminalExecutionsUnchanged(t *testing.T) {
	for _, state := range []execution.State{execution.StateCompleted, execution.StateFailed, execution.StateCancelled} {
		t.Run(string(state), func(t *testing.T) {
			f := newCancelNonterminalFixture(t)
			lockRunsWith(f.lock, openRow(), nil)
			f.store.On("FindByAssignment", mock.Anything, assignmentID).Return(terminalSnapshot(t, state), nil)

			err := f.op.Execute(context.Background(), accessEnded("access_revoked"))

			require.NoError(t, err)
			require.False(t, f.updated)
		})
	}
	t.Run("row finished between the read and the execution lock", func(t *testing.T) {
		f := newCancelNonterminalFixture(t)
		lockRunsWith(f.lock, openRow(), nil)
		f.store.On("FindByAssignment", mock.Anything, assignmentID).Return(snapshotIn(t, execution.StateRunning), nil)
		completed := terminalSnapshot(t, execution.StateCompleted)
		f.expectUpdate(completed)

		err := f.op.Execute(context.Background(), accessEnded("authority_expired"))

		require.NoError(t, err)
		require.Equal(t, completed, f.after)
		require.Nil(t, f.fact)
	})
}

// TestCancelNonterminalWithoutExecutionWritesNothing covers an assignment that never started.
func TestCancelNonterminalWithoutExecutionWritesNothing(t *testing.T) {
	f := newCancelNonterminalFixture(t)
	lockRunsWith(f.lock, openRow(), nil)
	f.store.On("FindByAssignment", mock.Anything, assignmentID).Return(execution.Snapshot{}, executionport.ErrNotFound)

	err := f.op.Execute(context.Background(), accessEnded("access_revoked"))

	require.NoError(t, err)
	require.False(t, f.updated)
}

// TestCancelNonterminalDoesNotRequireOpenAccess proves the caller's access decision is trusted, not re-read.
func TestCancelNonterminalDoesNotRequireOpenAccess(t *testing.T) {
	f := newCancelNonterminalFixture(t)
	revoked := openRow()
	revoked.AccessRevokedAt = timePtr(trustedNow().Add(-time.Minute))
	lockRunsWith(f.lock, revoked, nil)
	f.store.On("FindByAssignment", mock.Anything, assignmentID).Return(snapshotIn(t, execution.StateRunning), nil)
	f.expectUpdate(snapshotIn(t, execution.StateRunning))

	require.NoError(t, f.op.Execute(context.Background(), accessEnded("access_revoked")))
	require.Equal(t, execution.StateCancelled, f.after.State)
}

// TestCancelNonterminalRejectsBadShapeBeforeLocking covers malformed ids, actors and unsupported reasons.
func TestCancelNonterminalRejectsBadShapeBeforeLocking(t *testing.T) {
	mutations := map[string]func(*executionusecase.CancelNonterminalInput){
		"assignment id":      func(in *executionusecase.CancelNonterminalInput) { in.AssignmentID = "not-a-uuid" },
		"actor":              func(in *executionusecase.CancelNonterminalInput) { in.ActorRef = "" },
		"empty reason":       func(in *executionusecase.CancelNonterminalInput) { in.Reason = "" },
		"unsupported reason": func(in *executionusecase.CancelNonterminalInput) { in.Reason = "candidate_stopped" },
	}
	for name, mutate := range mutations {
		t.Run(name, func(t *testing.T) {
			f := newCancelNonterminalFixture(t)
			input := accessEnded("access_revoked")
			mutate(&input)

			err := f.op.Execute(context.Background(), input)

			requireCode(t, err, errs.InvalidInput)
		})
	}
}

// TestCancelNonterminalMapsFailures keeps a missing assignment public and other failures technical.
func TestCancelNonterminalMapsFailures(t *testing.T) {
	t.Run("missing assignment", func(t *testing.T) {
		f := newCancelNonterminalFixture(t)
		f.lock.On("WithAssignment", mock.Anything, assignmentID, mock.Anything).Return(executionport.ErrNotFound)

		requireCode(t, f.op.Execute(context.Background(), accessEnded("access_revoked")), errs.EXLNotFound)
	})
	t.Run("unexpected read failure", func(t *testing.T) {
		f := newCancelNonterminalFixture(t)
		lockRunsWith(f.lock, openRow(), nil)
		f.store.On("FindByAssignment", mock.Anything, assignmentID).Return(execution.Snapshot{}, errors.New("connection reset"))

		requireTechnical(t, f.op.Execute(context.Background(), accessEnded("access_revoked")))
	})
	t.Run("uncertain commit", func(t *testing.T) {
		f := newCancelNonterminalFixture(t)
		lockRunsWith(f.lock, openRow(), func(err error) error {
			require.NoError(t, err)
			return executionport.ErrUncertainCommit
		})
		f.store.On("FindByAssignment", mock.Anything, assignmentID).Return(snapshotIn(t, execution.StateCreated), nil)
		f.expectUpdate(snapshotIn(t, execution.StateCreated))

		requireCode(t, f.op.Execute(context.Background(), accessEnded("access_revoked")), errs.EXLUncertainCommit)
	})
}
