package interviewusecase_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/rfanazhari/amer-prototype/application/repository/interviewport"
	"github.com/rfanazhari/amer-prototype/application/service"
	"github.com/rfanazhari/amer-prototype/application/usecase/interviewusecase"
	"github.com/rfanazhari/amer-prototype/domain/aggregate/agentconfig"
	"github.com/rfanazhari/amer-prototype/domain/enum"
	"github.com/rfanazhari/amer-prototype/domain/interview"
	"github.com/rfanazhari/amer-prototype/domain/valueobject"
	"github.com/rfanazhari/amer-prototype/test/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// fixedTime supplies a deterministic instant with submillisecond precision to test normalization.
func fixedTime() time.Time {
	return time.Date(2026, 11, 2, 1, 0, 0, 123456789, time.FixedZone("fixture", 7*3600))
}

// pinnedRevision restores a fresh published revision without consulting current publication policy.
func pinnedRevision(t *testing.T) agentconfig.PublishedRevision[valueobject.HRConfiguration] {
	t.Helper()
	c, err := valueobject.NewHRConfiguration(valueobject.HRInput{
		MandatoryQuestions: []valueobject.QuestionInput{{ID: "background", Stage: "background", Text: "Experience?"}, {ID: "screening", Stage: "screening", Text: "Project?"}, {ID: "compensation", Stage: "compensation", Text: "Expectations?"}},
		StageSeconds:       &valueobject.StageSecondsInput{Background: 60, Screening: 240, Compensation: 90, Closing: 30},
		CompensationPolicy: &valueobject.CompensationInput{DiscussionMode: "expectations_only"},
	})
	require.NoError(t, err)
	v, err := agentconfig.NewDraftVersion(1)
	require.NoError(t, err)
	r, err := agentconfig.RestoreRevision(agentconfig.RevisionSnapshot[valueobject.HRConfiguration]{Identity: agentconfig.RevisionIdentity{ID: "revision", InstanceID: "instance", Number: 1, DraftVersion: v, Template: agentconfig.TemplateRef{Template: "hr", Version: "1", Behavior: "1"}}, ValidationID: "validation", Configuration: c, CreatedAt: fixedTime()})
	require.NoError(t, err)
	return r
}

// harness owns generated port mocks and a detached in-memory boundary for orchestration assertions.
// It simulates callback success/rollback, without asserting SQL or real lifecycle locking.
type harness struct {
	workflow                                                        interviewusecase.Workflow
	lifecycle                                                       *mocks.InterviewLifecycle
	store                                                           *mocks.ProgressStore
	driver                                                          *mocks.InterviewDriver
	summary                                                         *mocks.InterviewSummarizer
	record                                                          interviewport.ProgressRecord
	exists                                                          bool
	steps                                                           []string
	fenceErr, writeErr, commitErr, createErr, attachErr, summaryErr error
	activationCalls                                                 int
	budgetRequests                                                  []service.InterviewActivation
	creates                                                         []service.InterviewVoiceCreate
	contexts                                                        []service.InterviewVoiceContext
}

// authority supplies independent lifecycle authority; progress versions are tracked separately.
func authority() service.InterviewAttempt {
	return service.InterviewAttempt{ExecutionID: "execution", AttemptID: "old-attempt", ExpectedVersion: 7}
}

// request supplies a candidate session offer and pinned revision for the workflow boundary.
func request(t *testing.T, h *harness) interviewusecase.SessionRequest {
	t.Helper()
	return interviewusecase.SessionRequest{Authority: authority(), ProgressVersion: h.record.Version, SessionID: "new-session", IdempotencyKey: "key", SDP: "offer\n", Revision: pinnedRevision(t)}
}

// newHarness wires Mockery-generated ports and controlled time for each isolated test.
func newHarness(t *testing.T, covered bool) *harness {
	t.Helper()
	p, err := interview.NewProgress("execution", pinnedRevision(t), fixedTime())
	require.NoError(t, err)
	if covered {
		require.NoError(t, p.Ask("background"))
		require.NoError(t, p.RecordEntry(interview.TranscriptEntry{ID: "answer", Stage: enum.StageBackground, Speaker: interview.Candidate, State: interview.Final, Body: "Recorded answer"}))
		require.NoError(t, p.Answer("background", "answer"))
	}
	h := &harness{lifecycle: mocks.NewInterviewLifecycle(t), store: mocks.NewProgressStore(t), driver: mocks.NewInterviewDriver(t), summary: mocks.NewInterviewSummarizer(t), exists: true,
		record: interviewport.ProgressRecord{Snapshot: p.Snapshot(), Version: 3, Sessions: []interviewport.VoiceSession{{ID: "old-session", AttemptID: "old-attempt", Stage: enum.StageBackground, OpenedAt: fixedTime().UTC().Truncate(time.Millisecond)}}}}
	clock := mocks.NewClock(t)
	clock.On("Now").Return(fixedTime()).Maybe()
	h.workflow = interviewusecase.Workflow{Lifecycle: h.lifecycle, Store: h.store, Driver: h.driver, Clock: clock, Summary: h.summary}
	h.lifecycle.On("Fence", mock.Anything, mock.Anything).Return(func(_ context.Context, a service.InterviewAttempt) error {
		if h.fenceErr != nil {
			return h.fenceErr
		}
		if a.AttemptID == "stale" {
			return interview.ErrConflict
		}
		return nil
	}).Maybe()
	h.lifecycle.On("WithProgress", mock.Anything, mock.Anything, mock.Anything).Return(func(ctx context.Context, _ service.InterviewAttempt, write func(context.Context) error) error {
		if h.writeErr != nil {
			return h.writeErr
		}
		before := cloneRecord(t, h.record)
		err := write(context.WithValue(ctx, txKey{}, true))
		if err != nil || h.commitErr != nil {
			h.record = before
			if err != nil {
				return err
			}
			return h.commitErr
		}
		h.steps = append(h.steps, "commit")
		return nil
	}).Maybe()
	h.store.On("Find", mock.Anything, "execution").Return(func(context.Context, string) (interviewport.ProgressRecord, error) {
		if !h.exists {
			return interviewport.ProgressRecord{}, interviewport.ErrNotFound
		}
		return cloneRecord(t, h.record), nil
	}).Maybe()
	h.store.On("Update", mock.Anything, mock.Anything, mock.Anything).Return(func(ctx context.Context, w interviewport.ProgressWrite, change func(*interviewport.ProgressRecord) error) error {
		require.Equal(t, true, ctx.Value(txKey{}), "store must join lifecycle callback context")
		require.Equal(t, "execution", w.ExecutionID)
		if w.ExpectedVersion != h.record.Version {
			return interviewport.ErrConflict
		}
		next := cloneRecord(t, h.record)
		if err := change(&next); err != nil {
			return err
		}
		next.Version++
		h.record = next
		h.steps = append(h.steps, "update")
		return nil
	}).Maybe()
	h.store.On("Insert", mock.Anything, mock.Anything, mock.Anything).Return(func(ctx context.Context, _ interviewport.ProgressWrite, r interviewport.ProgressRecord) error {
		require.Equal(t, true, ctx.Value(txKey{}))
		h.record = r
		h.exists = true
		h.steps = append(h.steps, "insert")
		return nil
	}).Maybe()
	h.lifecycle.On("Activate", mock.Anything, mock.Anything).Return(func(_ context.Context, r service.InterviewActivation) (service.InterviewActivationResult, error) {
		h.activationCalls++
		h.budgetRequests = append(h.budgetRequests, r)
		if h.activationCalls > 2 {
			return service.InterviewActivationResult{}, interview.ErrBudget
		}
		a := r.Authority
		a.AttemptID = "replacement-attempt"
		a.ExpectedVersion++
		return service.InterviewActivationResult{Authority: a, RunningAt: fixedTime().UTC().Truncate(time.Millisecond), HardDeadline: fixedTime().UTC().Truncate(time.Millisecond).Add(15 * time.Minute)}, nil
	}).Maybe()
	h.driver.On("Create", mock.Anything, mock.Anything).Return(func(ctx context.Context, r service.InterviewVoiceCreate) (service.InterviewVoiceCreated, error) {
		require.NoError(t, ctx.Err())
		h.steps = append(h.steps, "create")
		h.creates = append(h.creates, r)
		s := r.Session
		s.ProviderSessionID = "provider-session"
		return service.InterviewVoiceCreated{Session: s, SDP: "answer"}, h.createErr
	}).Maybe()
	h.driver.On("Attach", mock.Anything, mock.Anything).Return(func(context.Context, service.InterviewVoiceSession) error { return h.attachErr }).Maybe()
	h.driver.On("SupplyContext", mock.Anything, mock.Anything).Return(func(_ context.Context, r service.InterviewVoiceContext) error {
		h.contexts = append(h.contexts, r)
		return nil
	}).Maybe()
	h.driver.On("Close", mock.Anything, mock.Anything).Return(nil).Maybe()
	h.summary.On("Summarize", mock.Anything, mock.Anything).Return(func(context.Context, interview.Handoff) (string, error) { return "optional narrative", h.summaryErr }).Maybe()
	return h
}

// txKey marks only mock lifecycle callback contexts, proving application context propagation.
type txKey struct{}

// cloneRecord detaches snapshots and mutable session fields in the mock storage boundary.
func cloneRecord(t *testing.T, r interviewport.ProgressRecord) interviewport.ProgressRecord {
	t.Helper()
	if r.Snapshot.ExecutionID != "" {
		p, err := interview.RestoreProgress(r.Snapshot)
		require.NoError(t, err)
		r.Snapshot = p.Snapshot()
	}
	r.Sessions = append([]interviewport.VoiceSession(nil), r.Sessions...)
	r.HandoffSummaries = cloneSummaries(r.HandoffSummaries)
	return r
}

// cloneSummaries returns a fresh optional summary map for simulated rollback.
func cloneSummaries(s map[enum.QuestionStage]string) map[enum.QuestionStage]string {
	if s == nil {
		return nil
	}
	result := make(map[enum.QuestionStage]string, len(s))
	for k, v := range s {
		result[k] = v
	}
	return result
}

// TestWorkflowPropose verifies authorization, evidence retention and durable-before-create ordering.
func TestWorkflowPropose(t *testing.T) {
	// proposalCase describes each public proposal outcome and its side effects.
	type proposalCase struct {
		name                             string
		covered                          bool
		target                           enum.QuestionStage
		summaryFails, commitFails, stale bool
		wantErr                          bool
	}
	tests := []proposalCase{
		{name: "uncovered writes nothing", target: enum.StageScreening, wantErr: true},
		{name: "cannot skip stage", covered: true, target: enum.StageCompensation, wantErr: true},
		{name: "stale attempt writes nothing", covered: true, target: enum.StageScreening, stale: true, wantErr: true},
		{name: "successful proposal", covered: true, target: enum.StageScreening},
		{name: "summary failure preserves evidence", covered: true, target: enum.StageScreening, summaryFails: true},
		{name: "commit failure prevents session", covered: true, target: enum.StageScreening, commitFails: true, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := newHarness(t, tt.covered)
			if tt.summaryFails {
				h.summaryErr = errors.New("summary unavailable")
			}
			if tt.commitFails {
				h.commitErr = errors.New("commit failed")
			}
			a := authority()
			if tt.stale {
				a.AttemptID = "stale"
			}
			before := cloneRecord(t, h.record)
			err := h.workflow.Propose(context.Background(), interviewusecase.Proposal{Authority: a, ProgressVersion: 3, Target: tt.target})
			if tt.wantErr {
				require.Error(t, err)
				assert.Equal(t, before, h.record)
				h.driver.AssertNotCalled(t, "Create", mock.Anything, mock.Anything)
				if !tt.commitFails {
					h.store.AssertNotCalled(t, "Update", mock.Anything, mock.Anything, mock.Anything)
				}
				return
			}
			require.NoError(t, err)
			h.driver.AssertNotCalled(t, "Create", mock.Anything, mock.Anything)
			require.Len(t, h.record.Snapshot.Handoffs, 1)
			assert.Equal(t, []string{"answer"}, h.record.Snapshot.Handoffs[0].EntryIDs)
			assert.Equal(t, interview.Answered, h.record.Snapshot.Handoffs[0].Coverage[0].State)
			assert.Equal(t, enum.StageBackground, h.record.Snapshot.CurrentStage)
			assert.Equal(t, fixedTime().UTC().Truncate(time.Millisecond), h.record.Snapshot.Handoffs[0].CommittedAt)
			require.NotNil(t, h.record.Sessions[0].ClosedAt)
			if tt.summaryFails {
				assert.Empty(t, h.record.HandoffSummaries)
			} else {
				assert.Equal(t, "optional narrative", h.record.HandoffSummaries[enum.StageBackground])
			}
			_, err = h.workflow.OpenStage(context.Background(), request(t, h))
			require.NoError(t, err)
			assert.Equal(t, []string{"update", "commit", "update", "commit", "create", "update", "commit"}, h.steps)
			assert.Equal(t, enum.StageScreening, h.record.Snapshot.CurrentStage)
			require.Len(t, h.record.Sessions, 2)
			assert.Equal(t, enum.StageScreening, h.record.Sessions[1].Stage)
		})
	}
}

// TestWorkflowActivation proves budgets are passed to the lifecycle and refusal never opens a third session.
func TestWorkflowActivation(t *testing.T) {
	// activationCase covers both successful transport and refused provider setup.
	type activationCase struct {
		name          string
		providerFails bool
		wantErr       bool
	}
	tests := []activationCase{{name: "success"}, {name: "provider fails", providerFails: true, wantErr: true}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := newHarness(t, false)
			h.exists = false
			if tt.providerFails {
				h.createErr = errors.New("provider unavailable")
			}
			r := request(t, h)
			r.ProgressVersion = 0
			created, err := h.workflow.Activate(context.Background(), r)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, "provider-session", created.Session.ProviderSessionID)
			}
			require.Len(t, h.budgetRequests, 1)
			assert.Equal(t, service.InterviewBudgets{ActivationAttempts: 2, ActivationAttemptTimeout: 30 * time.Second, ActivationWindow: 90 * time.Second, ReconnectWindow: 120 * time.Second, ReconnectAttempts: 2, ReconnectEpisodes: 2}, h.budgetRequests[0].Budgets)
			assert.Equal(t, "activate", h.budgetRequests[0].Intent)
			assert.Equal(t, "offer\r\n", h.creates[0].SDP)
			assert.Equal(t, "replacement-attempt", h.creates[0].Session.AttemptID)
			assert.Equal(t, 2, h.record.Snapshot.FollowUpLimit)
			if tt.providerFails {
				// Retry on the same persisted progress; the lifecycle owns the activation count.
				r.ProgressVersion = h.record.Version
				r.SessionID = "retry-session"
				r.IdempotencyKey = "retry-key"
				_, err = h.workflow.Activate(context.Background(), r)
				require.ErrorIs(t, err, h.createErr)
				r.ProgressVersion = h.record.Version
				r.SessionID = "third-session"
				r.IdempotencyKey = "third-key"
				_, err = h.workflow.Activate(context.Background(), r)
				require.ErrorIs(t, err, interview.ErrBudget)
				assert.Len(t, h.creates, 2)
				h.store.AssertNumberOfCalls(t, "Insert", 1)
			}
		})
	}
}

// TestWorkflowReconnect verifies backend detection, stale-attempt rejection and committed-context recovery.
func TestWorkflowReconnect(t *testing.T) {
	// reconnectCase varies stale authority and exhausted persisted recovery budgets.
	type reconnectCase struct {
		name                      string
		stale, exhausted, expired bool
		wantErr                   bool
	}
	tests := []reconnectCase{{name: "replace current session"}, {name: "drop old attempt", stale: true, wantErr: true}, {name: "third replacement refused", exhausted: true, wantErr: true}, {name: "120 second equality refused", expired: true, wantErr: true}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := newHarness(t, false)
			a := authority()
			if tt.stale {
				a.AttemptID = "stale"
			}
			session := service.InterviewVoiceSession{ExecutionID: "execution", AttemptID: a.AttemptID, SessionID: "old-session"}
			before := cloneRecord(t, h.record)
			err := h.workflow.Disconnect(context.Background(), a, session)
			if tt.stale {
				require.Error(t, err)
				assert.Equal(t, before, h.record)
				h.store.AssertNotCalled(t, "Update", mock.Anything, mock.Anything, mock.Anything)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, h.record.Snapshot.Reconnect.DetectedAt)
			assert.Equal(t, fixedTime().UTC().Truncate(time.Millisecond), *h.record.Snapshot.Reconnect.DetectedAt)
			if tt.exhausted {
				h.record.Snapshot.Reconnect.Attempts = 2
			}
			if tt.expired {
				at := fixedTime().UTC().Truncate(time.Millisecond).Add(-120 * time.Second)
				h.record.Snapshot.RunningAt = at
				h.record.Snapshot.StageActivatedAt = at
				h.record.Snapshot.Reconnect.DetectedAt = &at
			}
			p, restoreErr := interview.RestoreProgress(h.record.Snapshot)
			require.NoError(t, restoreErr)
			require.NoError(t, p.RecordEntry(interview.TranscriptEntry{ID: "committed", Stage: enum.StageBackground, Speaker: interview.Candidate, State: interview.Final, Body: "Checkpoint answer"}))
			require.NoError(t, p.RecordEntry(interview.TranscriptEntry{ID: "uncited", Stage: enum.StageBackground, Speaker: interview.Candidate, State: interview.Final, Body: "Outside checkpoint"}))
			h.record.Snapshot = p.Snapshot()
			h.record.Checkpoints = []interviewport.Checkpoint{{ID: "checkpoint", Stage: enum.StageBackground, SessionID: "old-session", AttemptID: "old-attempt", EntryIDs: []string{"committed"}, CommittedAt: fixedTime().UTC().Truncate(time.Millisecond)}}
			if tt.exhausted || tt.expired {
				h.lifecycle.On("Fail", mock.Anything, service.InterviewFailure{Authority: authority(), Reason: "reconnect_budget_exhausted"}).Return(nil).Once()
			}
			created, err := h.workflow.Reconnect(context.Background(), request(t, h))
			if tt.wantErr {
				require.Error(t, err)
				assert.Empty(t, h.creates)
				assert.Empty(t, h.budgetRequests)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, "execution", created.Session.ExecutionID)
			assert.Equal(t, "replacement-attempt", created.Session.AttemptID)
			assert.Equal(t, "new-session", created.Session.SessionID)
			require.Len(t, h.record.Sessions, 2)
			assert.Equal(t, "old-session", *h.record.Sessions[1].ReplacedSessionID)
			assert.Equal(t, 1, h.record.Snapshot.Reconnect.Episodes)
			assert.Equal(t, 1, h.record.Snapshot.Reconnect.Attempts)
			assert.Nil(t, h.record.Snapshot.Reconnect.DetectedAt)
			require.Len(t, h.contexts, 2)
			assert.Contains(t, h.contexts[0].Text, "Experience?")
			assert.JSONEq(t, `{"checkpointId":"checkpoint","entries":[{"ID":"committed","Stage":"background","Speaker":"candidate","State":"final","Body":"Checkpoint answer"}]}`, h.contexts[1].Text)
			assert.NotContains(t, h.contexts[1].Text, "Outside checkpoint")
			assert.Equal(t, "reconnect", h.budgetRequests[0].Intent)
			// After replacement even the former current attempt must be dropped by the lifecycle fence.
			h.fenceErr = interview.ErrConflict
			before = cloneRecord(t, h.record)
			err = h.workflow.Disconnect(context.Background(), authority(), session)
			require.Error(t, err)
			assert.Equal(t, before, h.record)
		})
	}
}
