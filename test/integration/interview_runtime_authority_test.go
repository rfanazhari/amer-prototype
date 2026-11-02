package integration_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/rfanazhari/amer-prototype/application/repository/interviewport"
	"github.com/rfanazhari/amer-prototype/application/service"
	"github.com/rfanazhari/amer-prototype/application/usecase/executionusecase"
	"github.com/rfanazhari/amer-prototype/application/usecase/interviewusecase"
	"github.com/rfanazhari/amer-prototype/domain/aggregate/execution"
	"github.com/rfanazhari/amer-prototype/domain/enum"
	"github.com/rfanazhari/amer-prototype/domain/interview"
	"github.com/rfanazhari/amer-prototype/infrastructure/repository/executionrepository"
	"github.com/rfanazhari/amer-prototype/infrastructure/repository/interviewrepository"
	"github.com/rfanazhari/amer-prototype/infrastructure/service/interviewlifecycle"
	"github.com/rfanazhari/amer-prototype/test/mocks"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// authorityFor constructs a fresh adapter to prove authority survives adapter and pool reloads.
func authorityFor(f progressFixture) interviewlifecycle.Lifecycle {
	executions := executionrepository.Store{DB: f.db}
	return interviewlifecycle.Lifecycle{Transactions: executions, Executions: executions, Attempts: executionrepository.Ledger{DB: f.db}, Commands: executions, Progress: f.store, Clock: f.store.Clock}
}

// admittedProgress persists an attempt before progress so real fences can guard the T10 fixture.
func admittedProgress(t *testing.T) (progressFixture, service.InterviewAttempt) {
	t.Helper()
	f := newProgressFixture(t)
	clock := mocks.NewClock(t)
	clock.On("Now").Return(f.running).Maybe()
	a := authorityFor(f)
	a.Clock = clock
	got, err := a.Activate(context.Background(), service.InterviewActivation{Authority: service.InterviewAttempt{ExecutionID: f.executionID, AttemptID: f.attemptID, ExpectedVersion: 1}, Intent: "activate", IdempotencyKey: "initial", Budgets: service.DefaultInterviewBudgets()})
	require.NoError(t, err)
	require.NoError(t, f.store.Insert(context.Background(), f.write(0), f.record(t)))
	return f, got.Authority
}

// TestInterviewRuntimeAuthorityPostgres proves durable generation, shared rollback and cancellation serialization.
func TestInterviewRuntimeAuthorityPostgres(t *testing.T) {
	t.Run("reload restores authority and rejected fences never write", func(t *testing.T) {
		f, a := admittedProgress(t)
		ctx := context.Background()
		adapter := authorityFor(f)
		stored, err := (executionrepository.Store{DB: f.db}).Find(ctx, f.executionID)
		require.NoError(t, err)
		require.Equal(t, int64(1), stored.CurrentGeneration)
		require.NoError(t, adapter.Fence(ctx, a))
		wrong := a
		wrong.AttemptID = uuid.NewString()
		stale := a
		stale.ExpectedVersion--
		for _, rejected := range []service.InterviewAttempt{wrong, stale} {
			called := false
			err := adapter.WithProgress(ctx, rejected, func(context.Context) error { called = true; return nil })
			require.Error(t, err)
			require.False(t, called)
		}
		require.NoError(t, f.db.Exec(`UPDATE agent_execution_attempts SET outcome = 'FAILED' WHERE id = ?`, f.attemptID).Error)
		second, err := adapter.Activate(ctx, service.InterviewActivation{Authority: service.InterviewAttempt{ExecutionID: f.executionID, AttemptID: uuid.NewString(), ExpectedVersion: 2}, Intent: "activate", IdempotencyKey: "replacement", Budgets: service.DefaultInterviewBudgets()})
		require.NoError(t, err)
		require.ErrorIs(t, adapter.Fence(ctx, a), interview.ErrConflict)
		oldAtCurrent := a
		oldAtCurrent.ExpectedVersion = second.Authority.ExpectedVersion
		require.ErrorIs(t, adapter.Fence(ctx, oldAtCurrent), interview.ErrConflict)
		reload := authorityFor(f)
		require.NoError(t, reload.Fence(ctx, second.Authority))
	})
	t.Run("callback failure rolls back execution command and progress writes", func(t *testing.T) {
		f, a := admittedProgress(t)
		adapter := authorityFor(f)
		ctx := context.Background()
		boom := errors.New("callback rollback")
		err := adapter.WithProgress(ctx, a, func(tx context.Context) error {
			require.NoError(t, f.store.Update(tx, f.write(1), func(r *interviewport.ProgressRecord) error { f.answerBackground(t, r); return nil }))
			command := interviewport.SessionCommand{ID: uuid.NewString(), ExecutionID: f.executionID, Key: "rollback", Intent: "activate", AttemptID: f.attemptID, SessionID: f.sessionID, State: "pending", ExecutionVersion: 2, CreatedAt: f.running}
			require.NoError(t, (interviewrepository.Commands{DB: f.db, Clock: f.store.Clock}).Insert(tx, command))
			_, err := (executionusecase.CommitProgress{Transactions: adapter.Transactions, Executions: adapter.Executions, Clock: adapter.Clock}).Execute(tx, executionusecase.CommitProgressInput{ExecutionID: f.executionID, ExpectedVersion: 2, Generation: 1, RequestKey: "probe", OperationKey: "initial", ActorRef: "probe", ProgressRef: f.executionID})
			require.NoError(t, err)
			return boom
		})
		require.ErrorIs(t, err, boom)
		require.EqualValues(t, 1, f.count(t, `SELECT count(*) FROM interview_progress WHERE execution_id = ? AND committed_version = 1`, f.executionID))
		require.Zero(t, f.count(t, `SELECT count(*) FROM interview_session_commands WHERE execution_id = ?`, f.executionID))
		require.Zero(t, f.count(t, `SELECT count(*) FROM interview_transcript_entries WHERE execution_id = ?`, f.executionID))
		snap, err := adapter.Executions.Find(ctx, f.executionID)
		require.NoError(t, err)
		require.Nil(t, snap.ProgressRef)
	})
	t.Run("cancel waits for progress lock then fences every later write", func(t *testing.T) {
		f, a := admittedProgress(t)
		adapter := authorityFor(f)
		ctx := context.Background()
		locked, release, writeDone := make(chan struct{}), make(chan struct{}), make(chan error, 1)
		go func() {
			writeDone <- adapter.WithProgress(ctx, a, func(tx context.Context) error {
				close(locked)
				<-release
				return f.store.Update(tx, f.write(1), func(r *interviewport.ProgressRecord) error { f.answerBackground(t, r); return nil })
			})
		}()
		<-locked
		cancelDone := make(chan error, 1)
		go func() {
			_, err := (executionusecase.Cancel{Transactions: adapter.Transactions, Executions: adapter.Executions, Commands: adapter.Commands, Clock: adapter.Clock}).Execute(ctx, executionusecase.CancelInput{ExecutionID: f.executionID, ExpectedVersion: 2, RequestKey: "cancel", ActorRef: "candidate", Reason: "candidate_cancelled"})
			cancelDone <- err
		}()
		close(release)
		require.NoError(t, <-writeDone)
		require.NoError(t, <-cancelDone)
		called := false
		err := adapter.WithProgress(ctx, a, func(context.Context) error { called = true; return nil })
		require.ErrorIs(t, err, interview.ErrConflict)
		require.False(t, called)
		require.EqualValues(t, 1, f.count(t, `SELECT count(*) FROM interview_transcript_entries WHERE execution_id = ?`, f.executionID))
	})
	t.Run("real handoff commits source close and rejected callback rolls back both", func(t *testing.T) {
		f, a := admittedProgress(t)
		ctx := context.Background()
		adapter := authorityFor(f)
		require.NoError(t, f.store.Update(ctx, f.write(1), func(r *interviewport.ProgressRecord) error { f.answerBackground(t, r); return nil }))
		workflow := interviewusecase.Workflow{Lifecycle: adapter, Store: f.store, Clock: f.store.Clock}
		boom := errors.New("reject handoff callback")
		err := adapter.WithProgress(ctx, a, func(tx context.Context) error {
			if err := workflow.Propose(tx, interviewusecase.Proposal{Authority: a, ProgressVersion: 2, Target: enum.StageScreening}); err != nil {
				return err
			}
			return boom
		})
		require.ErrorIs(t, err, boom)
		require.Zero(t, f.count(t, `SELECT count(*) FROM interview_handoffs WHERE execution_id = ?`, f.executionID))
		require.EqualValues(t, 1, f.count(t, `SELECT count(*) FROM interview_voice_sessions WHERE execution_id = ? AND closed_at IS NULL`, f.executionID))
		require.NoError(t, workflow.Propose(ctx, interviewusecase.Proposal{Authority: a, ProgressVersion: 2, Target: enum.StageScreening}))
		require.EqualValues(t, 1, f.count(t, `SELECT count(*) FROM interview_handoffs WHERE execution_id = ?`, f.executionID))
		require.Zero(t, f.count(t, `SELECT count(*) FROM interview_voice_sessions WHERE execution_id = ? AND closed_at IS NULL`, f.executionID))
		require.Zero(t, f.count(t, `SELECT count(*) FROM interview_voice_sessions WHERE execution_id = ? AND stage = 'screening'`, f.executionID))
		driver := mocks.NewInterviewDriver(t)
		calls := 0
		driver.On("Create", mock.Anything, mock.Anything).Return(func(_ context.Context, request service.InterviewVoiceCreate) (service.InterviewVoiceCreated, error) {
			calls++
			if calls == 1 {
				return service.InterviewVoiceCreated{}, errors.New("fake setup failure")
			}
			session := request.Session
			session.ProviderSessionID = "fake-target"
			return service.InterviewVoiceCreated{Session: session, SDP: "answer"}, nil
		})
		driver.On("Attach", mock.Anything, mock.Anything).Return(nil)
		driver.On("SupplyContext", mock.Anything, mock.Anything).Return(nil)
		workflow.Driver = driver
		request := interviewusecase.SessionRequest{Authority: a, ProgressVersion: 3, SessionID: uuid.NewString(), IdempotencyKey: "target-first", SDP: "offer"}
		_, err = workflow.OpenStage(ctx, request)
		require.Error(t, err)
		request.ProgressVersion = 4
		request.SessionID = uuid.NewString()
		request.IdempotencyKey = "target-retry"
		_, err = workflow.OpenStage(ctx, request)
		require.NoError(t, err)
		require.EqualValues(t, 1, f.count(t, `SELECT count(*) FROM interview_handoffs WHERE execution_id = ?`, f.executionID))
		require.EqualValues(t, 1, f.count(t, `SELECT count(*) FROM interview_voice_sessions WHERE execution_id = ? AND stage = 'screening' AND closed_at IS NULL AND provider_session_id IS NOT NULL`, f.executionID))

	})
	t.Run("hard deadline fails same execution retaining evidence", func(t *testing.T) {
		f, a := admittedProgress(t)
		ctx := context.Background()
		adapter := authorityFor(f)
		clock := mocks.NewClock(t)
		clock.On("Now").Return(f.running.Add(15 * time.Minute)).Maybe()
		adapter.Clock = clock
		require.ErrorIs(t, adapter.WithProgress(ctx, a, func(context.Context) error { return nil }), interview.ErrConflict)
		workflow := interviewusecase.Workflow{Lifecycle: adapter, Store: f.store, Clock: clock}
		require.NoError(t, workflow.Finish(ctx, interviewusecase.FinishRequest{Authority: a}))
		snap, err := adapter.Executions.Find(ctx, f.executionID)
		require.NoError(t, err)
		require.Equal(t, execution.StateFailed, snap.State)
		require.EqualValues(t, 1, f.count(t, `SELECT count(*) FROM interview_progress WHERE execution_id = ?`, f.executionID))
	})
}
