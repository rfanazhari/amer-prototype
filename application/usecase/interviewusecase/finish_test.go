package interviewusecase_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/rfanazhari/amer-prototype/application/repository/interviewport"
	"github.com/rfanazhari/amer-prototype/application/service"
	"github.com/rfanazhari/amer-prototype/application/usecase/interviewusecase"
	"github.com/rfanazhari/amer-prototype/domain/enum"
	"github.com/rfanazhari/amer-prototype/domain/interview"
	"github.com/rfanazhari/amer-prototype/test/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// finishRunning is the lifecycle running instant used by the completion fixtures.
func finishRunning() time.Time { return fixedTime().UTC().Truncate(time.Millisecond) }

// finishRecord builds progress through every stage, optionally leaving the closing handoff uncommitted.
func finishRecord(t *testing.T, closingSealed bool) interviewport.ProgressRecord {
	t.Helper()
	p, err := interview.NewProgress("execution", pinnedRevision(t), finishRunning())
	require.NoError(t, err)
	at := finishRunning().Add(time.Second)
	for _, step := range []struct {
		stage, next enum.QuestionStage
	}{{enum.StageBackground, enum.StageScreening}, {enum.StageScreening, enum.StageCompensation}, {enum.StageCompensation, enum.StageClosing}} {
		id := string(step.stage) + "-answer"
		require.NoError(t, p.Ask(string(step.stage)))
		require.NoError(t, p.RecordEntry(interview.TranscriptEntry{ID: id, Stage: step.stage, Speaker: interview.Candidate, State: interview.Final, Body: "Recorded answer"}))
		require.NoError(t, p.Answer(string(step.stage), id))
		require.NoError(t, p.CommitHandoff(step.next, at))
		require.NoError(t, p.ActivateStage(step.next, at))
	}
	if closingSealed {
		require.NoError(t, p.CommitHandoff("", at))
	}
	return interviewport.ProgressRecord{Snapshot: p.Snapshot(), Version: 9}
}

// TestWorkflowFinish verifies the single terminal outcome chosen by the required record and the hard deadline.
func TestWorkflowFinish(t *testing.T) {
	deadline := finishRunning().Add(15 * time.Minute)
	// finishCase selects the stored record, trusted instant and lifecycle behavior for one decision.
	type finishCase struct {
		name         string
		sealed       bool
		now          time.Time
		completeErr  error
		failErr      error
		wantComplete bool
		wantFail     bool
		wantErr      error
	}
	rejected := errors.New("lifecycle rejected")
	tests := []finishCase{
		{name: "required record completes once", sealed: true, now: finishRunning().Add(5 * time.Minute), wantComplete: true},
		{name: "required record at the exact deadline completes", sealed: true, now: deadline, wantComplete: true},
		{name: "missing record at the exact deadline fails", now: deadline, wantFail: true},
		{name: "missing record after the deadline fails", now: deadline.Add(time.Millisecond), wantFail: true},
		{name: "missing record before the deadline calls nothing", now: deadline.Add(-time.Millisecond), wantErr: interview.ErrConflict},
		{name: "complete rejection keeps rows", sealed: true, now: finishRunning().Add(time.Minute), completeErr: rejected, wantComplete: true, wantErr: rejected},
		{name: "fail rejection keeps rows", now: deadline, failErr: rejected, wantFail: true, wantErr: rejected},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			record := finishRecord(t, tt.sealed)
			stored := cloneRecord(t, record)
			lifecycle := mocks.NewInterviewLifecycle(t)
			store := mocks.NewProgressStore(t)
			clock := mocks.NewClock(t)
			clock.On("Now").Return(tt.now).Maybe()
			lifecycle.On("Fence", mock.Anything, authority()).Return(nil).Once()
			store.On("Find", mock.Anything, "execution").Return(record, nil).Once()
			if tt.wantComplete {
				lifecycle.On("Complete", mock.Anything, service.InterviewCompletion{Authority: authority(), ProgressReference: "execution"}).Return(tt.completeErr).Once()
			}
			if tt.wantFail {
				lifecycle.On("Fail", mock.Anything, mock.MatchedBy(func(r service.InterviewFailure) bool {
					return r.Authority == authority() && r.Reason != ""
				})).Return(tt.failErr).Once()
			}
			w := interviewusecase.Workflow{Lifecycle: lifecycle, Store: store, Clock: clock}

			err := w.Finish(context.Background(), interviewusecase.FinishRequest{Authority: authority()})

			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
			}
			assert.Equal(t, stored, cloneRecord(t, record))
			store.AssertNotCalled(t, "Update", mock.Anything, mock.Anything, mock.Anything)
			store.AssertNotCalled(t, "Insert", mock.Anything, mock.Anything, mock.Anything)
			lifecycle.AssertNotCalled(t, "WithProgress", mock.Anything, mock.Anything, mock.Anything)
			if !tt.wantComplete {
				lifecycle.AssertNotCalled(t, "Complete", mock.Anything, mock.Anything)
			}
			if !tt.wantFail {
				lifecycle.AssertNotCalled(t, "Fail", mock.Anything, mock.Anything)
			}
		})
	}
}

// TestWorkflowFinishRejectsBadAuthority verifies stale or incomplete authority reaches no terminal call.
func TestWorkflowFinishRejectsBadAuthority(t *testing.T) {
	// authorityCase selects an invalid authority or a fence rejection.
	type authorityCase struct {
		name      string
		authority service.InterviewAttempt
		fenceErr  error
		wantErr   error
	}
	tests := []authorityCase{
		{name: "missing attempt", authority: service.InterviewAttempt{ExecutionID: "execution", ExpectedVersion: 7}, wantErr: interview.ErrInvalid},
		{name: "stale attempt", authority: authority(), fenceErr: interview.ErrConflict, wantErr: interview.ErrConflict},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lifecycle := mocks.NewInterviewLifecycle(t)
			store := mocks.NewProgressStore(t)
			clock := mocks.NewClock(t)
			clock.On("Now").Return(finishRunning().Add(15 * time.Minute)).Maybe()
			lifecycle.On("Fence", mock.Anything, mock.Anything).Return(tt.fenceErr).Maybe()
			w := interviewusecase.Workflow{Lifecycle: lifecycle, Store: store, Clock: clock}

			err := w.Finish(context.Background(), interviewusecase.FinishRequest{Authority: tt.authority})

			assert.ErrorIs(t, err, tt.wantErr)
			lifecycle.AssertNotCalled(t, "Complete", mock.Anything, mock.Anything)
			lifecycle.AssertNotCalled(t, "Fail", mock.Anything, mock.Anything)
		})
	}
}
