package integration_test

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/rfanazhari/amer-prototype/application/dto"
	"github.com/rfanazhari/amer-prototype/application/repository/executionport"
	"github.com/rfanazhari/amer-prototype/application/service"
	"github.com/rfanazhari/amer-prototype/application/usecase/executionusecase"
	"github.com/rfanazhari/amer-prototype/domain/aggregate/execution"
	"github.com/rfanazhari/amer-prototype/test/mocks"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// lifecycleActor is the opaque actor token every operation in these proofs reports.
const lifecycleActor = "runtime:t8"

// lifecycleOps wires the real execution operations to one PostgreSQL store.
type lifecycleOps struct {
	start     executionusecase.Start
	admit     executionusecase.Admit
	activate  executionusecase.Activate
	progress  executionusecase.CommitProgress
	complete  executionusecase.Complete
	cancel    executionusecase.Cancel
	cancelEnd executionusecase.CancelNonterminal
	committed *mocks.ProgressCommitment
}

// newLifecycleOps builds every operation on the fixture store; the progress port is a Mockery mock answering true.
func newLifecycleOps(t *testing.T, f executionFixture) lifecycleOps {
	t.Helper()
	clock := mocks.NewClock(t)
	clock.On("Now").Return(time.Now()).Maybe()
	committed := mocks.NewProgressCommitment(t)
	committed.On("IsCommitted", mock.Anything, mock.Anything, mock.Anything).Return(true, nil).Maybe()
	s := f.store
	return lifecycleOps{
		start:     executionusecase.Start{Assignments: s, Executions: s, Commands: commandsOf(f), Clock: clock},
		admit:     executionusecase.Admit{Transactions: s, Executions: s, Commands: commandsOf(f), Clock: clock},
		activate:  executionusecase.Activate{Transactions: s, Executions: s, Commands: commandsOf(f), Clock: clock},
		progress:  executionusecase.CommitProgress{Transactions: s, Executions: s, Clock: clock},
		complete:  executionusecase.Complete{Transactions: s, Executions: s, Commands: commandsOf(f), Progress: committed, Clock: clock},
		cancel:    executionusecase.Cancel{Transactions: s, Executions: s, Commands: commandsOf(f), Clock: clock},
		cancelEnd: executionusecase.CancelNonterminal{Assignments: s, Executions: s, Clock: clock},
		committed: committed,
	}
}

// commandsOf returns the command store of the fixture's PostgreSQL adapter.
func commandsOf(f executionFixture) executionport.CommandStore { return f.store }

// startInput builds a start command for the fixture assignment.
func (f executionFixture) startInput(key string, refs ...string) executionusecase.StartInput {
	return executionusecase.StartInput{AssignmentID: f.assignmentID, RequestKey: key, ActorRef: lifecycleActor, InputRefs: refs}
}

// admitInput builds a pinless admit command for the fixture binding.
func (f executionFixture) admitInput(key string) executionusecase.AdmitInput {
	return executionusecase.AdmitInput{
		Scope: "doc-" + key, RequestKey: key, ActorRef: lifecycleActor,
		InstanceID: f.binding.InstanceID, RevisionID: f.binding.RevisionID,
	}
}

// runToProgress activates an admitted execution and commits progress, returning the RUNNING view.
func (o lifecycleOps) runToProgress(t *testing.T, created *dto.ExecutionView) *dto.ExecutionView {
	t.Helper()
	ctx := context.Background()
	running, err := o.activate.Execute(ctx, executionusecase.ActivateInput{
		ExecutionID: created.ID, RequestKey: "act-" + created.ID[:8], ActorRef: lifecycleActor,
		ExpectedVersion: created.StateVersion, OperationKey: "op-1", Generation: 1,
	})
	require.NoError(t, err)
	require.Equal(t, string(execution.StateRunning), running.State)
	progressed, err := o.progress.Execute(ctx, executionusecase.CommitProgressInput{
		ExecutionID: created.ID, RequestKey: "prog-" + created.ID[:8], ActorRef: lifecycleActor,
		ExpectedVersion: running.StateVersion, OperationKey: "op-1", Generation: 1, ProgressRef: "progress-1",
	})
	require.NoError(t, err)
	require.Equal(t, string(execution.StateRunning), progressed.State)
	require.Equal(t, "progress-1", progressed.ProgressRef)
	return progressed
}

// completeInput builds the completion command that matches a progressed view.
func completeInput(v *dto.ExecutionView) executionusecase.CompleteInput {
	return executionusecase.CompleteInput{
		ExecutionID: v.ID, RequestKey: "done-" + v.ID[:8], ActorRef: lifecycleActor,
		ExpectedVersion: v.StateVersion, OperationKey: "op-1", Generation: 1, ProgressRef: v.ProgressRef,
	}
}

// cancelInput builds the cancel command that matches a progressed view.
func cancelInput(v *dto.ExecutionView) executionusecase.CancelInput {
	return executionusecase.CancelInput{
		ExecutionID: v.ID, RequestKey: "stop-" + v.ID[:8], ActorRef: lifecycleActor,
		ExpectedVersion: v.StateVersion, Reason: "candidate_stopped",
	}
}

// lifecycleCase names one operation-level PostgreSQL proof.
type lifecycleCase struct {
	name   string
	verify func(t *testing.T, f executionFixture, ops lifecycleOps)
}

// TestExecutionLifecyclePostgres proves locking, rollback and idempotency of the lifecycle operations on PostgreSQL.
func TestExecutionLifecyclePostgres(t *testing.T) {
	tests := []lifecycleCase{
		{name: "concurrent starts leave one execution", verify: proveConcurrentStarts},
		{name: "lost-response retry returns the same execution", verify: proveLostResponseRetry},
		{name: "conflicting intent does not replace the execution", verify: proveConflictingIntent},
		{name: "failed command write rolls back shared row and pin", verify: proveStartRollback},
		{name: "completion versus cancellation leaves one terminal state", verify: proveCompleteCancelRace},
		{name: "later active revision does not change the stored binding", verify: proveFrozenBinding},
		{name: "admit completes with no pin row", verify: proveAdmitWithoutPin},
		{name: "access ended cancels once and never reopens", verify: proveAccessEndedCancel},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newExecutionFixture(t)
			tt.verify(t, f, newLifecycleOps(t, f))
		})
	}
}

// proveConcurrentStarts races starts with distinct and repeated keys and expects one execution and one pin.
func proveConcurrentStarts(t *testing.T, f executionFixture, ops lifecycleOps) {
	const workers = 8
	views := make([]*dto.ExecutionView, workers)
	errs := make([]error, workers)
	var wg sync.WaitGroup
	for i := range workers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			views[i], errs[i] = ops.start.Execute(context.Background(), f.startInput(fmt.Sprintf("key-%d", i%3)))
		}()
	}
	wg.Wait()
	for i := range workers {
		require.NoError(t, errs[i])
		require.Equal(t, views[0].ID, views[i].ID)
		require.Equal(t, string(execution.StateCreated), views[i].State)
		require.Equal(t, f.binding.RevisionID, views[i].RevisionID)
	}
	require.EqualValues(t, 1, f.count(t, `SELECT count(*) FROM interview_executions WHERE assignment_id = ?`, f.assignmentID))
	require.EqualValues(t, 1, f.count(t, `SELECT count(*) FROM agent_executions WHERE id = ?`, views[0].ID))
	require.EqualValues(t, 1, f.count(t, `SELECT count(*) FROM agent_execution_transitions WHERE execution_id = ?`, views[0].ID))
	require.EqualValues(t, 3, f.count(t, `SELECT count(*) FROM agent_execution_commands WHERE execution_id = ?`, views[0].ID))
}

// proveLostResponseRetry repeats the same start after its response was lost and expects the same row.
func proveLostResponseRetry(t *testing.T, f executionFixture, ops lifecycleOps) {
	first, err := ops.start.Execute(context.Background(), f.startInput("retry-1", "doc-a"))
	require.NoError(t, err)
	again, err := ops.start.Execute(context.Background(), f.startInput("retry-1", "doc-a"))
	require.NoError(t, err)
	require.Equal(t, first, again)
	byAssignment, err := f.store.FindByAssignment(context.Background(), f.assignmentID)
	require.NoError(t, err)
	require.Equal(t, first.ID, byAssignment.ID)
	require.EqualValues(t, 1, f.count(t, `SELECT count(*) FROM agent_executions WHERE instance_id = ?`, f.binding.InstanceID))
}

// proveConflictingIntent reuses a key with another input set and expects a conflict that writes nothing.
func proveConflictingIntent(t *testing.T, f executionFixture, ops lifecycleOps) {
	first, err := ops.start.Execute(context.Background(), f.startInput("conflict-1", "doc-a"))
	require.NoError(t, err)
	_, err = ops.start.Execute(context.Background(), f.startInput("conflict-1", "doc-b"))
	require.ErrorIs(t, err, service.ErrExecutionIdempotencyConflict)
	stored, err := f.store.FindByAssignment(context.Background(), f.assignmentID)
	require.NoError(t, err)
	require.Equal(t, first.ID, stored.ID)
	require.EqualValues(t, 1, f.count(t, `SELECT count(*) FROM agent_executions WHERE instance_id = ?`, f.binding.InstanceID))
	require.EqualValues(t, 1, f.count(t, `SELECT count(*) FROM agent_execution_commands WHERE execution_id = ?`, first.ID))
}

// failingCommands writes the real command row and then reports a failure so the surrounding transaction must roll back.
type failingCommands struct{ executionport.CommandStore }

// FindOrInsert stores the command through the real store, then fails.
func (c failingCommands) FindOrInsert(ctx context.Context, scope, kind, key, fingerprint, executionID string) (string, error) {
	if _, err := c.CommandStore.FindOrInsert(ctx, scope, kind, key, fingerprint, executionID); err != nil {
		return "", err
	}
	return "", errRollbackProbe
}

// proveStartRollback fails after the execution, transition, pin and command writes and expects none to persist.
func proveStartRollback(t *testing.T, f executionFixture, ops lifecycleOps) {
	ops.start.Commands = failingCommands{CommandStore: f.store}
	_, err := ops.start.Execute(context.Background(), f.startInput("rollback-1"))
	require.Error(t, err)
	require.Zero(t, f.count(t, `SELECT count(*) FROM interview_executions WHERE assignment_id = ?`, f.assignmentID))
	require.Zero(t, f.count(t, `SELECT count(*) FROM agent_executions WHERE instance_id = ?`, f.binding.InstanceID))
	require.Zero(t, f.count(t, `SELECT count(*) FROM agent_execution_commands WHERE scope = ?`, f.assignmentID))

	view, err := ops.start.Execute(context.Background(), f.startInput("rollback-1"))
	require.NoError(t, err)
	require.Equal(t, string(execution.StateCreated), view.State)
}

// proveCompleteCancelRace runs completion against cancellation on fresh executions and expects exactly one winner each time.
func proveCompleteCancelRace(t *testing.T, f executionFixture, ops lifecycleOps) {
	for round := range 6 {
		created, err := ops.admit.Execute(context.Background(), f.admitInput(fmt.Sprintf("race-%d", round)))
		require.NoError(t, err)
		progressed := ops.runToProgress(t, created)

		var wg sync.WaitGroup
		var completed, cancelled *dto.ExecutionView
		var completeErr, cancelErr error
		wg.Add(2)
		go func() {
			defer wg.Done()
			completed, completeErr = ops.complete.Execute(context.Background(), completeInput(progressed))
		}()
		go func() {
			defer wg.Done()
			cancelled, cancelErr = ops.cancel.Execute(context.Background(), cancelInput(progressed))
		}()
		wg.Wait()

		require.True(t, (completeErr == nil) != (cancelErr == nil), "exactly one terminal command may win: complete=%v cancel=%v", completeErr, cancelErr)
		stored, err := f.store.Find(context.Background(), created.ID)
		require.NoError(t, err)
		if completeErr == nil {
			require.Equal(t, execution.StateCompleted, stored.State)
			require.Equal(t, completed.State, string(stored.State))
		} else {
			require.Equal(t, execution.StateCancelled, stored.State)
			require.Equal(t, cancelled.State, string(stored.State))
		}
		require.EqualValues(t, 1, f.count(t, `SELECT count(*) FROM agent_execution_transitions WHERE execution_id = ? AND to_state IN ('COMPLETED','CANCELLED','FAILED')`, created.ID))
	}
}

// proveFrozenBinding publishes a second revision after start and expects the stored execution to keep the first.
func proveFrozenBinding(t *testing.T, f executionFixture, ops lifecycleOps) {
	created, err := ops.start.Execute(context.Background(), f.startInput("frozen-1"))
	require.NoError(t, err)

	configStore, _ := acpStore(t)
	replaceDocumentDraft(t, configStore, f.binding.InstanceID, 1, []string{"total", "currency"})
	secondValidation := insertOwnedValidation(t, configStore, f.binding.InstanceID, 2, true)
	var accountID string
	require.NoError(t, f.db.Raw(`SELECT created_by FROM candidates c JOIN interview_assignments a ON a.candidate_id = c.id WHERE a.id = ?`, f.assignmentID).Scan(&accountID).Error)
	secondRevision := publishOwned(t, configStore, f.binding.InstanceID, secondValidation, accountID, uuid.NewString(), "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb2")
	require.NotEqual(t, f.binding.RevisionID, secondRevision)

	again, err := ops.start.Execute(context.Background(), f.startInput("frozen-1"))
	require.NoError(t, err)
	require.Equal(t, created.ID, again.ID)
	require.Equal(t, f.binding.RevisionID, again.RevisionID)
	stored, err := f.store.Find(context.Background(), created.ID)
	require.NoError(t, err)
	require.Equal(t, f.binding, stored.Binding)
}

// proveAdmitWithoutPin runs an admitted execution to COMPLETED and expects no interview pin to exist for it.
func proveAdmitWithoutPin(t *testing.T, f executionFixture, ops lifecycleOps) {
	created, err := ops.admit.Execute(context.Background(), f.admitInput("pinless-1"))
	require.NoError(t, err)
	require.Empty(t, created.AssignmentID)
	progressed := ops.runToProgress(t, created)
	done, err := ops.complete.Execute(context.Background(), completeInput(progressed))
	require.NoError(t, err)
	require.Equal(t, string(execution.StateCompleted), done.State)
	require.Zero(t, f.count(t, `SELECT count(*) FROM interview_executions WHERE id = ?`, created.ID))
	again, err := ops.complete.Execute(context.Background(), completeInput(progressed))
	require.NoError(t, err)
	require.Equal(t, done.State, again.State)
	require.Equal(t, done.StateVersion, again.StateVersion)
}

// proveAccessEndedCancel revokes the assignment, cancels the nonterminal execution twice and tries to start again.
func proveAccessEndedCancel(t *testing.T, f executionFixture, ops lifecycleOps) {
	ctx := context.Background()
	created, err := ops.start.Execute(ctx, f.startInput("revoke-1"))
	require.NoError(t, err)
	require.NoError(t, f.db.Exec(`UPDATE interview_assignments SET access_revoked_at = now() WHERE id = ?`, f.assignmentID).Error)

	input := executionusecase.CancelNonterminalInput{AssignmentID: f.assignmentID, ActorRef: lifecycleActor, Reason: "access_revoked"}
	require.NoError(t, ops.cancelEnd.Execute(ctx, input))
	require.NoError(t, ops.cancelEnd.Execute(ctx, input))
	stored, err := f.store.Find(ctx, created.ID)
	require.NoError(t, err)
	require.Equal(t, execution.StateCancelled, stored.State)
	require.Equal(t, "access_revoked", stored.TerminalReason)
	version := stored.StateVersion

	_, err = ops.start.Execute(ctx, f.startInput("revoke-2"))
	require.ErrorIs(t, err, service.ErrExecutionAssignmentUnavailable)
	after, err := f.store.Find(ctx, created.ID)
	require.NoError(t, err)
	require.Equal(t, execution.StateCancelled, after.State)
	require.Equal(t, version, after.StateVersion)
	require.EqualValues(t, 1, f.count(t, `SELECT count(*) FROM agent_executions WHERE instance_id = ?`, f.binding.InstanceID))
}
