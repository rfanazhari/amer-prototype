package integration_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/rfanazhari/amer-prototype/application/repository/interviewport"
	"github.com/rfanazhari/amer-prototype/application/service"
	"github.com/rfanazhari/amer-prototype/application/usecase/interviewusecase"
	"github.com/rfanazhari/amer-prototype/domain/aggregate/execution"
	"github.com/rfanazhari/amer-prototype/domain/enum"
	"github.com/rfanazhari/amer-prototype/domain/interview"
	"github.com/rfanazhari/amer-prototype/test/mocks"
	"github.com/stretchr/testify/require"
)

// sealRequiredRecord builds every required handoff through public domain behavior and the real joined store.
func sealRequiredRecord(t *testing.T, f progressFixture, a service.InterviewAttempt) {
	t.Helper()
	ctx := context.Background()
	adapter := authorityFor(f)
	version := int64(1)
	// stageStep carries the fixed successor and question identity for one stored stage.
	type stageStep struct {
		stage    enum.QuestionStage
		next     enum.QuestionStage
		question string
	}
	steps := []stageStep{{enum.StageBackground, enum.StageScreening, "background"}, {enum.StageScreening, enum.StageCompensation, "screening"}, {enum.StageCompensation, enum.StageClosing, "compensation"}, {enum.StageClosing, "", ""}}
	for i, step := range steps {
		at := f.running.Add(time.Duration(i+1) * time.Minute)
		require.NoError(t, adapter.WithProgress(ctx, a, func(tx context.Context) error {
			return f.store.Update(tx, f.write(version), func(r *interviewport.ProgressRecord) error {
				p, err := interview.RestoreProgress(r.Snapshot)
				if err != nil {
					return err
				}
				if step.question != "" {
					entry := uuid.NewString()
					if err = p.Ask(step.question); err != nil {
						return err
					}
					if err = p.RecordEntry(interview.TranscriptEntry{ID: entry, Stage: step.stage, Speaker: interview.Candidate, State: interview.Final, Body: "committed evidence"}); err != nil {
						return err
					}
					if err = p.Answer(step.question, entry); err != nil {
						return err
					}
				}
				if err = p.CommitHandoff(step.next, at); err != nil {
					return err
				}
				r.Snapshot = p.Snapshot()
				r.Sessions[len(r.Sessions)-1].ClosedAt = &at
				return nil
			})
		}))
		version++
		if step.next == "" {
			continue
		}
		require.NoError(t, adapter.WithProgress(ctx, a, func(tx context.Context) error {
			return f.store.Update(tx, f.write(version), func(r *interviewport.ProgressRecord) error {
				p, err := interview.RestoreProgress(r.Snapshot)
				if err != nil {
					return err
				}
				if err = p.ActivateStage(step.next, at); err != nil {
					return err
				}
				r.Snapshot = p.Snapshot()
				r.Sessions = append(r.Sessions, interviewport.VoiceSession{ID: uuid.NewString(), AttemptID: a.AttemptID, Stage: step.next, OpenedAt: at})
				return nil
			})
		}))
		version++
	}
}

// TestInterviewRuntimeTerminalPostgres exercises real complete/fail operations without reports or invented retry budgets.
func TestInterviewRuntimeTerminalPostgres(t *testing.T) {
	t.Run("required closing record completes through real lifecycle", func(t *testing.T) {
		f, a := admittedProgress(t)
		sealRequiredRecord(t, f, a)
		adapter := authorityFor(f)
		workflow := interviewusecase.Workflow{Lifecycle: adapter, Store: f.store, Clock: f.store.Clock}
		require.NoError(t, workflow.Finish(context.Background(), interviewusecase.FinishRequest{Authority: a}))
		snap, err := adapter.Executions.Find(context.Background(), f.executionID)
		require.NoError(t, err)
		require.Equal(t, execution.StateCompleted, snap.State)
		require.Equal(t, execution.ProgressRef(f.executionID), *snap.ProgressRef)
		require.EqualValues(t, 4, f.count(t, `SELECT count(*) FROM interview_handoffs WHERE execution_id = ?`, f.executionID))
	})
	t.Run("initial window equality commits failure outside preparation rollback", func(t *testing.T) {
		f, a := admittedProgress(t)
		ctx := context.Background()
		adapter := authorityFor(f)
		clock := mocks.NewClock(t)
		clock.On("Now").Return(f.running.Add(90 * time.Second)).Maybe()
		adapter.Clock = clock
		workflow := interviewusecase.Workflow{Transactions: adapter.Transactions, Lifecycle: adapter, Store: f.store, Clock: clock}
		_, err := workflow.Activate(ctx, interviewusecase.SessionRequest{Authority: service.InterviewAttempt{ExecutionID: f.executionID, AttemptID: uuid.NewString(), ExpectedVersion: a.ExpectedVersion}, ProgressVersion: 1, SessionID: uuid.NewString(), IdempotencyKey: "expired", SDP: "offer"})
		require.ErrorIs(t, err, interview.ErrBudget)
		snap, err := adapter.Executions.Find(ctx, f.executionID)
		require.NoError(t, err)
		require.Equal(t, execution.StateFailed, snap.State)
		require.EqualValues(t, 1, f.count(t, `SELECT count(*) FROM agent_execution_attempts WHERE execution_id = ?`, f.executionID))
	})
	t.Run("reconnect exhaustion fails without provider work", func(t *testing.T) {
		f, a := admittedProgress(t)
		ctx := context.Background()
		adapter := authorityFor(f)
		require.NoError(t, f.store.Update(ctx, f.write(1), func(r *interviewport.ProgressRecord) error {
			p, err := interview.RestoreProgress(r.Snapshot)
			if err != nil {
				return err
			}
			if err = p.BeginReconnect(f.running); err != nil {
				return err
			}
			if err = p.AttemptReconnect(f.running); err != nil {
				return err
			}
			if err = p.AttemptReconnect(f.running); err != nil {
				return err
			}
			r.Snapshot = p.Snapshot()
			return nil
		}))
		workflow := interviewusecase.Workflow{Transactions: adapter.Transactions, Lifecycle: adapter, Store: f.store, Clock: f.store.Clock}
		_, err := workflow.Reconnect(ctx, interviewusecase.SessionRequest{Authority: service.InterviewAttempt{ExecutionID: f.executionID, AttemptID: uuid.NewString(), ExpectedVersion: a.ExpectedVersion}, ProgressVersion: 2, SessionID: uuid.NewString(), IdempotencyKey: "exhausted", SDP: "offer"})
		require.ErrorIs(t, err, interview.ErrBudget)
		snap, err := adapter.Executions.Find(ctx, f.executionID)
		require.NoError(t, err)
		require.Equal(t, execution.StateFailed, snap.State)
	})
	t.Run("third reconnect episode fails the same execution", func(t *testing.T) {
		f, a := admittedProgress(t)
		ctx := context.Background()
		adapter := authorityFor(f)
		require.NoError(t, f.store.Update(ctx, f.write(1), func(r *interviewport.ProgressRecord) error {
			p, err := interview.RestoreProgress(r.Snapshot)
			if err != nil {
				return err
			}
			for range 2 {
				if err = p.BeginReconnect(f.running); err != nil {
					return err
				}
				if err = p.AttemptReconnect(f.running); err != nil {
					return err
				}
				if err = p.FinishReconnect(); err != nil {
					return err
				}
			}
			r.Snapshot = p.Snapshot()
			return nil
		}))
		workflow := interviewusecase.Workflow{Lifecycle: adapter, Store: f.store, Clock: f.store.Clock}
		err := workflow.Disconnect(ctx, a, service.InterviewVoiceSession{ExecutionID: f.executionID, AttemptID: f.attemptID, SessionID: f.sessionID})
		require.ErrorIs(t, err, interview.ErrBudget)
		snap, err := adapter.Executions.Find(ctx, f.executionID)
		require.NoError(t, err)
		require.Equal(t, execution.StateFailed, snap.State)
		record, err := f.store.Find(ctx, f.executionID)
		require.NoError(t, err)
		require.Equal(t, 2, record.Snapshot.Reconnect.Episodes)
	})

}
