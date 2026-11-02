package interviewusecase_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/rfanazhari/amer-prototype/application/repository/interviewport"
	"github.com/rfanazhari/amer-prototype/application/service"
	"github.com/rfanazhari/amer-prototype/application/usecase/interviewusecase"
	"github.com/rfanazhari/amer-prototype/domain/aggregate/candidateinterview"
	"github.com/rfanazhari/amer-prototype/domain/interview"
	"github.com/rfanazhari/amer-prototype/test/mocks"
	"github.com/rfanazhari/amer-prototype/test/testdata"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// readyHarness owns the ports of one readiness scenario and the progress record the mock store mutates.
type readyHarness struct {
	op          interviewusecase.Ready
	current     *mocks.CurrentCandidateSession
	executions  *mocks.ExecutionLifecycle
	controllers *mocks.Controllers
	lifecycle   *mocks.InterviewLifecycle
	progress    *mocks.ProgressStore
	record      interviewport.ProgressRecord
}

// newReadyHarness builds a progress record whose open session for attempt-1 has provider identity but no readiness.
func newReadyHarness(t *testing.T, provider bool) *readyHarness {
	t.Helper()
	p, err := interview.NewProgress(attemptExecution, pinnedRevision(t), fixedTime())
	require.NoError(t, err)
	session := interviewport.VoiceSession{ID: "session-1", AttemptID: "attempt-1", Stage: p.Snapshot().CurrentStage, OpenedAt: fixedTime().UTC()}
	if provider {
		id := "provider-1"
		session.ProviderSessionID = &id
	}
	h := &readyHarness{
		current: mocks.NewCurrentCandidateSession(t), executions: mocks.NewExecutionLifecycle(t), controllers: mocks.NewControllers(t),
		lifecycle: mocks.NewInterviewLifecycle(t), progress: mocks.NewProgressStore(t),
		record: interviewport.ProgressRecord{Snapshot: p.Snapshot(), Version: 2, Sessions: []interviewport.VoiceSession{session}},
	}
	clock := mocks.NewClock(t)
	clock.On("Now").Return(fixedTime()).Maybe()
	h.op = interviewusecase.Ready{Current: h.current, Executions: h.executions, Controllers: h.controllers, Lifecycle: h.lifecycle, Progress: h.progress, Clock: clock}
	return h
}

// authorize makes the session own the execution and the tab the current controller.
func (h *readyHarness) authorize(t *testing.T, execution, tab string) {
	t.Helper()
	h.current.On("Resolve", mock.Anything).Return(testdata.CandidateSessionRecord(t), nil)
	h.executions.On("Find", mock.Anything, testdata.AssignmentID).Return(service.ExecutionFacts{ID: execution, State: "RUNNING", Version: 5}, nil)
	if execution != attemptExecution {
		return
	}
	id, err := candidateinterview.ParseControllerID(tab)
	require.NoError(t, err)
	c, err := candidateinterview.ClaimController(testdata.AssignmentID, attemptExecution, id, fixedTime())
	require.NoError(t, err)
	h.controllers.On("Find", mock.Anything, testdata.AssignmentID).Return(c.Snapshot(), nil)
}

// commit runs the lifecycle callback against the mocked store, which applies the change to h.record.
func (h *readyHarness) commit(fence error) {
	authority := service.InterviewAttempt{ExecutionID: attemptExecution, AttemptID: "attempt-1", ExpectedVersion: 5}
	h.lifecycle.On("Fence", mock.Anything, authority).Return(fence)
	if fence != nil {
		return
	}
	h.progress.On("Find", mock.Anything, attemptExecution).Return(func(context.Context, string) interviewport.ProgressRecord { return h.record }, nil)
	h.lifecycle.On("WithProgress", mock.Anything, authority, mock.Anything).Return(func(ctx context.Context, _ service.InterviewAttempt, write func(context.Context) error) error {
		return write(ctx)
	})
	h.progress.On("Update", mock.Anything, interviewport.ProgressWrite{ExecutionID: attemptExecution, AttemptID: "attempt-1", ExpectedVersion: 2}, mock.Anything).Return(
		func(_ context.Context, _ interviewport.ProgressWrite, change func(*interviewport.ProgressRecord) error) error {
			next := h.record
			next.Sessions = append([]interviewport.VoiceSession(nil), h.record.Sessions...)
			if err := change(&next); err != nil {
				return err
			}
			h.record = next
			return nil
		})
}

// TestReadyExecute covers ownership, controller, fencing, the readiness flag and untouched coverage.
func TestReadyExecute(t *testing.T) {
	// testCase is one readiness scenario with its arrangement and expected outcome.
	type testCase struct {
		name     string
		provider bool
		attempt  string
		arrange  func(*testing.T, *readyHarness)
		want     *interviewusecase.ReadyResult
		wantErr  error
	}
	tests := []testCase{
		{
			name: "records readiness and reports ready with provider identity", provider: true, attempt: "attempt-1",
			arrange: func(t *testing.T, h *readyHarness) { h.authorize(t, attemptExecution, attemptTab); h.commit(nil) },
			want:    &interviewusecase.ReadyResult{AttemptID: "attempt-1", SessionID: "session-1", ExecutionVersion: 5, SessionCondition: "ready"},
		},
		{
			name: "records readiness but stays pending without provider identity", attempt: "attempt-1",
			arrange: func(t *testing.T, h *readyHarness) { h.authorize(t, attemptExecution, attemptTab); h.commit(nil) },
			want:    &interviewusecase.ReadyResult{AttemptID: "attempt-1", SessionID: "session-1", ExecutionVersion: 5, SessionCondition: "pending"},
		},
		{
			name: "execution of another assignment is not found", attempt: "attempt-1",
			arrange: func(t *testing.T, h *readyHarness) {
				h.authorize(t, "99999999-9999-4999-8999-999999999999", attemptTab)
			},
			wantErr: service.ErrInterviewNotFound,
		},
		{
			name: "controller that is not current conflicts", attempt: "attempt-1",
			arrange: func(t *testing.T, h *readyHarness) { h.authorize(t, attemptExecution, "tab-2") },
			wantErr: interviewusecase.ErrAttemptConflict,
		},
		{
			name: "superseded attempt is not ready", attempt: "attempt-1",
			arrange: func(t *testing.T, h *readyHarness) {
				h.authorize(t, attemptExecution, attemptTab)
				h.commit(interview.ErrConflict)
			},
			wantErr: interviewusecase.ErrAttemptNotReady,
		},
		{
			name: "current attempt without an open session is not ready", attempt: "attempt-1",
			arrange: func(t *testing.T, h *readyHarness) {
				h.authorize(t, attemptExecution, attemptTab)
				h.record.Sessions[0].AttemptID = "attempt-0"
				h.commit(nil)
			},
			wantErr: interviewusecase.ErrAttemptNotReady,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := newReadyHarness(t, tt.provider)
			tt.arrange(t, h)
			before := h.record.Snapshot
			got, err := h.op.Execute(context.Background(), interviewusecase.ReadyInput{ExecutionID: attemptExecution, AttemptID: tt.attempt, ControllerID: attemptTab, ExpectedVersion: 5})
			if tt.wantErr != nil {
				require.Error(t, err)
				assert.True(t, errors.Is(err, tt.wantErr), "got %v", err)
				assert.Nil(t, got)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
			require.NotNil(t, h.record.Sessions[0].ReadinessAt)
			assert.Equal(t, fixedTime().UTC().Truncate(time.Millisecond), *h.record.Sessions[0].ReadinessAt)
			assert.Equal(t, before, h.record.Snapshot, "readiness must not change coverage, stage or transcript")
		})
	}
}
