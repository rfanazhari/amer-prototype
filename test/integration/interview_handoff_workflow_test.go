package integration_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/rfanazhari/amer-prototype/application/repository/interviewport"
	"github.com/rfanazhari/amer-prototype/application/service"
	"github.com/rfanazhari/amer-prototype/application/usecase/interviewusecase"
	"github.com/rfanazhari/amer-prototype/domain/enum"
	"github.com/rfanazhari/amer-prototype/test/mocks"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// handoffWorkflow wires the interview workflow to the real store with a pass-through lifecycle double.
type handoffWorkflow struct {
	workflow  interviewusecase.Workflow
	fenceErr  *error
	createErr *error
	authority service.InterviewAttempt
}

// newHandoffWorkflow seeds an answered background stage and wires mocked lifecycle and driver ports.
// The lifecycle double only forwards the callback; it proves no cross-aggregate fence.
func newHandoffWorkflow(t *testing.T, f progressFixture) handoffWorkflow {
	t.Helper()
	ctx := context.Background()
	require.NoError(t, f.store.Insert(ctx, f.write(0), f.record(t)))
	loaded, err := f.store.Find(ctx, f.executionID)
	require.NoError(t, err)
	f.answerBackground(t, &loaded)
	require.NoError(t, f.store.Update(ctx, f.write(1), func(r *interviewport.ProgressRecord) error { *r = loaded; return nil }))

	var fenceErr, createErr error
	lifecycle := mocks.NewInterviewLifecycle(t)
	lifecycle.On("Fence", mock.Anything, mock.Anything).Return(func(context.Context, service.InterviewAttempt) error { return fenceErr }).Maybe()
	lifecycle.On("WithProgress", mock.Anything, mock.Anything, mock.Anything).Return(func(ctx context.Context, _ service.InterviewAttempt, write func(context.Context) error) error {
		return write(ctx)
	}).Maybe()
	driver := mocks.NewInterviewDriver(t)
	driver.On("Create", mock.Anything, mock.Anything).Return(func(_ context.Context, r service.InterviewVoiceCreate) (service.InterviewVoiceCreated, error) {
		s := r.Session
		s.ProviderSessionID = "provider-" + uuid.NewString()
		return service.InterviewVoiceCreated{Session: s, SDP: "answer"}, createErr
	}).Maybe()
	driver.On("Attach", mock.Anything, mock.Anything).Return(nil).Maybe()
	driver.On("SupplyContext", mock.Anything, mock.Anything).Return(nil).Maybe()
	driver.On("Close", mock.Anything, mock.Anything).Return(nil).Maybe()
	return handoffWorkflow{
		workflow:  interviewusecase.Workflow{Lifecycle: lifecycle, Store: f.store, Driver: driver, Clock: f.store.Clock},
		fenceErr:  &fenceErr,
		createErr: &createErr,
		authority: service.InterviewAttempt{ExecutionID: f.executionID, AttemptID: f.attemptID, ExpectedVersion: 1},
	}
}

// TestInterviewHandoffWorkflowPostgres proves handoff commit, deferred target rows and target-only retry on PostgreSQL.
func TestInterviewHandoffWorkflowPostgres(t *testing.T) {
	tests := []progressStoreCase{
		{name: "handoff and source close commit together and no target session row exists", verify: proveHandoffWithoutTarget},
		{name: "fence rejection leaves progress, handoffs and sessions unchanged", verify: proveHandoffFenceRejection},
		{name: "failed target create retries the same target on the same execution", verify: proveTargetRetry},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) { tt.verify(t, newProgressFixture(t)) })
	}
}

// proveHandoffWithoutTarget commits the handoff and checks only the source session row exists.
func proveHandoffWithoutTarget(t *testing.T, f progressFixture) {
	w := newHandoffWorkflow(t, f)

	require.NoError(t, w.workflow.Propose(context.Background(), interviewusecase.Proposal{Authority: w.authority, ProgressVersion: 2, Target: enum.StageScreening}))

	require.EqualValues(t, 1, f.count(t, `SELECT count(*) FROM interview_handoffs WHERE execution_id = ? AND source_stage = 'background' AND target_stage = 'screening'`, f.executionID))
	require.EqualValues(t, 1, f.count(t, `SELECT count(*) FROM interview_voice_sessions WHERE execution_id = ?`, f.executionID))
	require.EqualValues(t, 0, f.count(t, `SELECT count(*) FROM interview_voice_sessions WHERE execution_id = ? AND closed_at IS NULL`, f.executionID))
	require.EqualValues(t, 0, f.count(t, `SELECT count(*) FROM interview_voice_sessions WHERE execution_id = ? AND stage = 'screening'`, f.executionID))
}

// proveHandoffFenceRejection shows a rejected fence writes no handoff and does not bump the version.
func proveHandoffFenceRejection(t *testing.T, f progressFixture) {
	w := newHandoffWorkflow(t, f)
	*w.fenceErr = errors.New("superseded attempt")

	err := w.workflow.Propose(context.Background(), interviewusecase.Proposal{Authority: w.authority, ProgressVersion: 2, Target: enum.StageScreening})

	require.Error(t, err)
	require.EqualValues(t, 0, f.count(t, `SELECT count(*) FROM interview_handoffs WHERE execution_id = ?`, f.executionID))
	require.EqualValues(t, 1, f.count(t, `SELECT count(*) FROM interview_voice_sessions WHERE execution_id = ? AND closed_at IS NULL`, f.executionID))
	require.EqualValues(t, 1, f.count(t, `SELECT count(*) FROM interview_progress WHERE execution_id = ? AND committed_version = 2 AND current_stage = 'background'`, f.executionID))
}

// proveTargetRetry fails the first target create, then retries and finds one open provisioned target session.
func proveTargetRetry(t *testing.T, f progressFixture) {
	ctx := context.Background()
	w := newHandoffWorkflow(t, f)
	require.NoError(t, w.workflow.Propose(ctx, interviewusecase.Proposal{Authority: w.authority, ProgressVersion: 2, Target: enum.StageScreening}))
	*w.createErr = errors.New("provider failed")
	req := interviewusecase.SessionRequest{Authority: w.authority, ProgressVersion: 3, SessionID: uuid.NewString(), IdempotencyKey: "first", SDP: "offer"}
	_, err := w.workflow.OpenStage(ctx, req)
	require.Error(t, err)
	*w.createErr = nil

	req.ProgressVersion, req.SessionID, req.IdempotencyKey = 4, uuid.NewString(), "retry"
	_, err = w.workflow.OpenStage(ctx, req)

	require.NoError(t, err)
	require.EqualValues(t, 1, f.count(t, `SELECT count(*) FROM interview_handoffs WHERE execution_id = ?`, f.executionID))
	require.EqualValues(t, 1, f.count(t, `SELECT count(*) FROM interview_voice_sessions WHERE execution_id = ? AND stage = 'screening' AND closed_at IS NULL AND provider_session_id IS NOT NULL`, f.executionID))
}
