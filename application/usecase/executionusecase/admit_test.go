package executionusecase_test

import (
	"context"
	"strings"
	"testing"

	"github.com/rfanazhari/amer-prototype/application/repository/executionport"
	"github.com/rfanazhari/amer-prototype/application/usecase/executionusecase"
	"github.com/rfanazhari/amer-prototype/domain/aggregate/execution"
	"github.com/rfanazhari/amer-prototype/pkg/errs"
	"github.com/rfanazhari/amer-prototype/test/mocks"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// admitFixture wires Admit to strict mocks. It has no assignment port by construction.
type admitFixture struct {
	tx       *mocks.Transactor
	store    *mocks.ExecutionStore
	commands *mocks.CommandStore
	admit    executionusecase.Admit
}

// newAdmitFixture builds an Admit whose transaction runs the callback and returns its error.
func newAdmitFixture(t *testing.T, wrap func(error) error) admitFixture {
	t.Helper()
	f := admitFixture{tx: mocks.NewTransactor(t), store: mocks.NewExecutionStore(t), commands: mocks.NewCommandStore(t)}
	f.tx.On("WithTransaction", mock.Anything, mock.Anything).Return(
		func(ctx context.Context, fn func(context.Context) error) error {
			err := fn(ctx)
			if wrap != nil {
				return wrap(err)
			}
			return err
		}).Maybe()
	f.admit = executionusecase.Admit{Transactions: f.tx, Executions: f.store, Commands: f.commands, Clock: testClock(t), NewID: fixedID(newID)}
	return f
}

// validAdmit is a well-formed admit command for a non-interview scope.
func validAdmit() executionusecase.AdmitInput {
	return executionusecase.AdmitInput{Scope: "doc-run:42", RequestKey: "admit-001", ActorRef: "analyzer", InstanceID: instanceID, RevisionID: revisionID, InputRefs: []string{"file-1"}}
}

// TestAdmitCommitsCreatedExecutionWithoutPin covers a first admit: CREATED, caller binding, no assignment id.
func TestAdmitCommitsCreatedExecutionWithoutPin(t *testing.T) {
	f := newAdmitFixture(t, nil)
	var gotSnap execution.Snapshot
	f.store.On("Insert", mock.Anything, mock.Anything, mock.Anything).Return(
		func(_ context.Context, snap execution.Snapshot, _ execution.TransitionFact) error {
			gotSnap = snap
			return nil
		})
	f.commands.On("FindOrInsert", mock.Anything, "doc-run:42", "start", "admit-001", mock.Anything, newID).Return(newID, nil)

	view, err := f.admit.Execute(context.Background(), validAdmit())

	require.NoError(t, err)
	require.NotNil(t, view)
	require.Equal(t, newID, view.ID)
	require.Empty(t, view.AssignmentID)
	require.Equal(t, "CREATED", view.State)
	require.Equal(t, instanceID, view.InstanceID)
	require.Equal(t, revisionID, view.RevisionID)
	require.Nil(t, view.ActivatedAt)
	require.Equal(t, execution.StateCreated, gotSnap.State)
	require.Equal(t, trustedNow(), gotSnap.CreatedAt)
}

// TestAdmitReplayReturnsStoredExecutionAndRollsBackTheDuplicate proves a repeated key does not leave a second row.
func TestAdmitReplayReturnsStoredExecutionAndRollsBackTheDuplicate(t *testing.T) {
	var callbackErr error
	f := newAdmitFixture(t, func(err error) error { callbackErr = err; return err })
	f.store.On("Insert", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	f.commands.On("FindOrInsert", mock.Anything, "doc-run:42", "start", "admit-001", mock.Anything, newID).Return(existingID, nil)
	f.store.On("Find", mock.Anything, existingID).Return(snapshotIn(t, execution.StateRunning), nil)

	view, err := f.admit.Execute(context.Background(), validAdmit())

	require.NoError(t, err)
	require.NotNil(t, view)
	require.Equal(t, existingID, view.ID)
	require.Equal(t, "RUNNING", view.State)
	require.Error(t, callbackErr, "the callback must abort so the duplicate insert rolls back")
}

// TestAdmitFaults covers a conflicting intent, an uncertain commit and a missing revision.
func TestAdmitFaults(t *testing.T) {
	t.Run("different fingerprint conflicts", func(t *testing.T) {
		f := newAdmitFixture(t, nil)
		f.store.On("Insert", mock.Anything, mock.Anything, mock.Anything).Return(nil)
		f.commands.On("FindOrInsert", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return("", executionport.ErrIdempotencyConflict)

		view, err := f.admit.Execute(context.Background(), validAdmit())

		require.Nil(t, view)
		requireCode(t, err, errs.EXLIdempotencyConflict)
	})
	t.Run("uncertain commit returns no id", func(t *testing.T) {
		f := newAdmitFixture(t, func(err error) error {
			require.NoError(t, err)
			return executionport.ErrUncertainCommit
		})
		f.store.On("Insert", mock.Anything, mock.Anything, mock.Anything).Return(nil)
		f.commands.On("FindOrInsert", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(newID, nil)

		view, err := f.admit.Execute(context.Background(), validAdmit())

		require.Nil(t, view)
		requireCode(t, err, errs.EXLUncertainCommit)
	})
	t.Run("missing revision is dependency unavailable", func(t *testing.T) {
		f := newAdmitFixture(t, nil)
		f.store.On("Insert", mock.Anything, mock.Anything, mock.Anything).Return(executionport.ErrDependencyUnavailable)

		view, err := f.admit.Execute(context.Background(), validAdmit())

		require.Nil(t, view)
		requireCode(t, err, errs.EXLDependencyUnavailable)
	})
}

// TestAdmitRejectsMalformedInputBeforeAnyWrite proves shape errors open no transaction.
func TestAdmitRejectsMalformedInputBeforeAnyWrite(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*executionusecase.AdmitInput)
	}{
		{"empty scope", func(in *executionusecase.AdmitInput) { in.Scope = "" }},
		{"scope with slash", func(in *executionusecase.AdmitInput) { in.Scope = "a/b" }},
		{"scope too long", func(in *executionusecase.AdmitInput) { in.Scope = strings.Repeat("s", 65) }},
		{"bad key", func(in *executionusecase.AdmitInput) { in.RequestKey = "bad key" }},
		{"bad actor", func(in *executionusecase.AdmitInput) { in.ActorRef = "" }},
		{"instance not a uuid", func(in *executionusecase.AdmitInput) { in.InstanceID = "x" }},
		{"revision not a uuid", func(in *executionusecase.AdmitInput) { in.RevisionID = "" }},
		{"bad input ref", func(in *executionusecase.AdmitInput) { in.InputRefs = []string{"no good"} }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newAdmitFixture(t, nil)
			in := validAdmit()
			tt.mutate(&in)

			view, err := f.admit.Execute(context.Background(), in)

			require.Nil(t, view)
			requireCode(t, err, errs.InvalidInput)
		})
	}
}
