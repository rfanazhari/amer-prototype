package interviewusecase_test

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/rfanazhari/amer-prototype/application/repository/interviewport"
	"github.com/rfanazhari/amer-prototype/application/service"
	"github.com/rfanazhari/amer-prototype/application/usecase/interviewusecase"
	"github.com/rfanazhari/amer-prototype/domain/aggregate/candidateinterview"
	"github.com/rfanazhari/amer-prototype/domain/enum"
	"github.com/rfanazhari/amer-prototype/domain/interview"
	"github.com/rfanazhari/amer-prototype/test/mocks"
	"github.com/rfanazhari/amer-prototype/test/testdata"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

const (
	attemptExecution = "22222222-2222-4222-8222-222222222222"
	attemptTab       = "tab-1"
	attemptKey       = "attempt-key-1"
)

// attemptHarness owns the Mockery ports of one session-attempt scenario.
type attemptHarness struct {
	op          interviewusecase.Attempts
	current     *mocks.CurrentCandidateSession
	executions  *mocks.ExecutionLifecycle
	controllers *mocks.Controllers
	commands    *mocks.SessionCommands
	revisions   *mocks.RevisionReader
	progress    *mocks.ProgressStore
	sessions    *mocks.SessionStarter
}

// newAttemptHarness wires strict mocks, a controlled clock and deterministic identities.
func newAttemptHarness(t *testing.T) *attemptHarness {
	t.Helper()
	h := &attemptHarness{
		current: mocks.NewCurrentCandidateSession(t), executions: mocks.NewExecutionLifecycle(t), controllers: mocks.NewControllers(t),
		commands: mocks.NewSessionCommands(t), revisions: mocks.NewRevisionReader(t), progress: mocks.NewProgressStore(t), sessions: mocks.NewSessionStarter(t),
	}
	clock := mocks.NewClock(t)
	clock.On("Now").Return(fixedTime()).Maybe()
	ids := 0
	h.op = interviewusecase.Attempts{
		Current: h.current, Executions: h.executions, Controllers: h.controllers, Commands: h.commands, Revisions: h.revisions,
		Progress: h.progress, Sessions: h.sessions, Clock: clock,
		NewID: func() string { ids++; return fmt.Sprintf("id-%d", ids) },
	}
	return h
}

// session makes the candidate session resolve to the fixture assignment.
func (h *attemptHarness) session(t *testing.T) {
	t.Helper()
	h.current.On("Resolve", mock.Anything).Return(testdata.CandidateSessionRecord(t), nil)
}

// owned makes the assignment's execution the one in the path at version 4.
func (h *attemptHarness) owned(version int64, id string) *mock.Call {
	return h.executions.On("Find", mock.Anything, testdata.AssignmentID).Return(service.ExecutionFacts{ID: id, AssignmentID: testdata.AssignmentID, State: "RUNNING", Version: version}, nil)
}

// controlled makes tab the current controller.
func (h *attemptHarness) controlled(t *testing.T, tab string) {
	t.Helper()
	id, err := candidateinterview.ParseControllerID(tab)
	require.NoError(t, err)
	c, err := candidateinterview.ClaimController(testdata.AssignmentID, attemptExecution, id, fixedTime())
	require.NoError(t, err)
	h.controllers.On("Find", mock.Anything, testdata.AssignmentID).Return(c.Snapshot(), nil)
}

// storedKey is the replay key scoped to the candidate session.
func storedKey() string { return testdata.CandidateSessionID + ":" + attemptKey }

// created is the provider result of a successful session create.
func created(attempt, session string) service.InterviewVoiceCreated {
	return service.InterviewVoiceCreated{
		Session: service.InterviewVoiceSession{ExecutionID: attemptExecution, AttemptID: attempt, SessionID: session, ProviderSessionID: "provider-1"},
		SDP:     "answer-sdp",
	}
}

// attemptInput is a valid activation request.
func attemptInput() interviewusecase.AttemptInput {
	return interviewusecase.AttemptInput{ExecutionID: attemptExecution, ControllerID: attemptTab, IdempotencyKey: attemptKey, Intent: "activate", SDP: "offer", ExpectedVersion: 4}
}

// TestAttemptsExecute covers authorization order, replay, intent conflict, provider ordering and error mapping.
func TestAttemptsExecute(t *testing.T) {
	type testCase struct {
		name    string
		mutate  func(*interviewusecase.AttemptInput)
		arrange func(*testing.T, *attemptHarness)
		want    *interviewusecase.AttemptResult
		wantErr error
	}
	tests := []testCase{
		{
			name: "activate creates attempt and records command",
			arrange: func(t *testing.T, h *attemptHarness) {
				h.session(t)
				h.owned(4, attemptExecution).Once()
				h.controlled(t, attemptTab)
				h.commands.On("Find", mock.Anything, attemptExecution, storedKey()).Return(interviewport.SessionCommand{}, interviewport.ErrNotFound)
				h.revisions.On("Find", mock.Anything, attemptExecution).Return(pinnedRevision(t), nil)
				h.progress.On("Find", mock.Anything, attemptExecution).Return(interviewport.ProgressRecord{}, interviewport.ErrNotFound)
				h.sessions.On("Activate", mock.Anything, mock.MatchedBy(func(r interviewusecase.SessionRequest) bool {
					return r.Authority == service.InterviewAttempt{ExecutionID: attemptExecution, AttemptID: "id-1", ExpectedVersion: 4} &&
						r.SessionID == "id-2" && r.IdempotencyKey == storedKey() && r.SDP == "offer" && r.Revision.Snapshot().Identity.ID == "revision"
				})).Return(created("id-1", "id-2"), nil)
				h.owned(5, attemptExecution).Once()
				h.commands.On("Insert", mock.Anything, mock.MatchedBy(func(c interviewport.SessionCommand) bool {
					return c.ExecutionID == attemptExecution && c.Key == storedKey() && c.Intent == "activate" && c.AttemptID == "id-1" &&
						c.SessionID == "id-2" && c.AnswerSDP == "answer-sdp" && c.ExecutionVersion == 5 && c.CreatedAt.Equal(fixedTime().UTC().Truncate(1e6))
				})).Return(nil)
			},
			want: &interviewusecase.AttemptResult{AttemptID: "id-1", SessionID: "id-2", SDP: "answer-sdp", ExecutionVersion: 5, SessionCondition: "pending"},
		},
		{
			name:   "reconnect passes the stored progress version without reading the revision",
			mutate: func(in *interviewusecase.AttemptInput) { in.Intent = "reconnect" },
			arrange: func(t *testing.T, h *attemptHarness) {
				h.session(t)
				h.owned(4, attemptExecution).Once()
				h.controlled(t, attemptTab)
				h.commands.On("Find", mock.Anything, attemptExecution, storedKey()).Return(interviewport.SessionCommand{}, interviewport.ErrNotFound)
				h.progress.On("Find", mock.Anything, attemptExecution).Return(interviewport.ProgressRecord{Version: 3}, nil)
				h.sessions.On("Reconnect", mock.Anything, mock.MatchedBy(func(r interviewusecase.SessionRequest) bool {
					return r.ProgressVersion == 3 && r.Authority.AttemptID == "id-1" && r.SessionID == "id-2"
				})).Return(created("id-1", "id-2"), nil)
				h.owned(5, attemptExecution).Once()
				h.commands.On("Insert", mock.Anything, mock.MatchedBy(func(c interviewport.SessionCommand) bool { return c.Intent == "reconnect" })).Return(nil)
			},
			want: &interviewusecase.AttemptResult{AttemptID: "id-1", SessionID: "id-2", SDP: "answer-sdp", ExecutionVersion: 5, SessionCondition: "pending"},
		},
		{
			name: "same key and intent replays the original answer without a provider call",
			arrange: func(t *testing.T, h *attemptHarness) {
				h.session(t)
				h.owned(5, attemptExecution)
				h.controlled(t, attemptTab)
				h.commands.On("Find", mock.Anything, attemptExecution, storedKey()).Return(interviewport.SessionCommand{
					ExecutionID: attemptExecution, Key: storedKey(), Intent: "activate", AttemptID: "id-1", SessionID: "id-2", AnswerSDP: "answer-sdp", ExecutionVersion: 5,
				}, nil)
				provider := "provider-1"
				ready := fixedTime()
				h.progress.On("Find", mock.Anything, attemptExecution).Return(interviewport.ProgressRecord{Sessions: []interviewport.VoiceSession{
					{ID: "id-2", AttemptID: "id-1", Stage: enum.StageBackground, ProviderSessionID: &provider, ReadinessAt: &ready},
				}}, nil)
			},
			want: &interviewusecase.AttemptResult{AttemptID: "id-1", SessionID: "id-2", SDP: "answer-sdp", ExecutionVersion: 5, SessionCondition: "ready", Replayed: true},
		},
		{
			name: "same key with another intent conflicts and creates nothing",
			arrange: func(t *testing.T, h *attemptHarness) {
				h.session(t)
				h.owned(5, attemptExecution)
				h.controlled(t, attemptTab)
				h.commands.On("Find", mock.Anything, attemptExecution, storedKey()).Return(interviewport.SessionCommand{Intent: "reconnect", AttemptID: "id-1"}, nil)
			},
			wantErr: interviewusecase.ErrAttemptConflict,
		},
		{
			name: "execution of another assignment is not found",
			arrange: func(t *testing.T, h *attemptHarness) {
				h.session(t)
				h.owned(4, "99999999-9999-4999-8999-999999999999")
			},
			wantErr: service.ErrInterviewNotFound,
		},
		{
			name: "controller that is not current conflicts",
			arrange: func(t *testing.T, h *attemptHarness) {
				h.session(t)
				h.owned(4, attemptExecution)
				h.controlled(t, "tab-2")
			},
			wantErr: interviewusecase.ErrAttemptConflict,
		},
		{
			name: "stale execution version conflicts before any provider call",
			arrange: func(t *testing.T, h *attemptHarness) {
				h.session(t)
				h.owned(6, attemptExecution)
				h.controlled(t, attemptTab)
				h.commands.On("Find", mock.Anything, attemptExecution, storedKey()).Return(interviewport.SessionCommand{}, interviewport.ErrNotFound)
			},
			wantErr: interviewusecase.ErrAttemptConflict,
		},
		{
			name: "exhausted budget is a conflict and stores no command",
			arrange: func(t *testing.T, h *attemptHarness) {
				h.session(t)
				h.owned(4, attemptExecution)
				h.controlled(t, attemptTab)
				h.commands.On("Find", mock.Anything, attemptExecution, storedKey()).Return(interviewport.SessionCommand{}, interviewport.ErrNotFound)
				h.revisions.On("Find", mock.Anything, attemptExecution).Return(pinnedRevision(t), nil)
				h.progress.On("Find", mock.Anything, attemptExecution).Return(interviewport.ProgressRecord{}, interviewport.ErrNotFound)
				h.sessions.On("Activate", mock.Anything, mock.Anything).Return(service.InterviewVoiceCreated{}, fmt.Errorf("wrapped: %w", interview.ErrBudget))
			},
			wantErr: interviewusecase.ErrAttemptConflict,
		},
		{
			name: "provider failure stays classified and stores no command",
			arrange: func(t *testing.T, h *attemptHarness) {
				h.session(t)
				h.owned(4, attemptExecution)
				h.controlled(t, attemptTab)
				h.commands.On("Find", mock.Anything, attemptExecution, storedKey()).Return(interviewport.SessionCommand{}, interviewport.ErrNotFound)
				h.revisions.On("Find", mock.Anything, attemptExecution).Return(pinnedRevision(t), nil)
				h.progress.On("Find", mock.Anything, attemptExecution).Return(interviewport.ProgressRecord{}, interviewport.ErrNotFound)
				h.sessions.On("Activate", mock.Anything, mock.Anything).Return(service.InterviewVoiceCreated{}, interviewusecase.ErrProviderUnavailable)
			},
			wantErr: interviewusecase.ErrProviderUnavailable,
		},
		{
			name: "unbound candidate session passes through",
			arrange: func(t *testing.T, h *attemptHarness) {
				h.current.On("Resolve", mock.Anything).Return(testdata.CandidateSessionRecord(t), service.ErrCandidateSessionUnbound)
			},
			wantErr: service.ErrCandidateSessionUnbound,
		},
		{name: "unknown intent is invalid", mutate: func(in *interviewusecase.AttemptInput) { in.Intent = "restart" }, arrange: func(*testing.T, *attemptHarness) {}, wantErr: interviewusecase.ErrAttemptInvalid},
		{name: "blank offer is invalid", mutate: func(in *interviewusecase.AttemptInput) { in.SDP = "  \n" }, arrange: func(*testing.T, *attemptHarness) {}, wantErr: interviewusecase.ErrAttemptInvalid},
		{name: "missing key is invalid", mutate: func(in *interviewusecase.AttemptInput) { in.IdempotencyKey = "" }, arrange: func(*testing.T, *attemptHarness) {}, wantErr: interviewusecase.ErrAttemptInvalid},
		{name: "missing version is invalid", mutate: func(in *interviewusecase.AttemptInput) { in.ExpectedVersion = 0 }, arrange: func(*testing.T, *attemptHarness) {}, wantErr: interviewusecase.ErrAttemptInvalid},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := newAttemptHarness(t)
			tt.arrange(t, h)
			in := attemptInput()
			if tt.mutate != nil {
				tt.mutate(&in)
			}
			got, err := h.op.Execute(context.Background(), in)
			if tt.wantErr != nil {
				require.Error(t, err)
				assert.True(t, errors.Is(err, tt.wantErr), "got %v", err)
				assert.Nil(t, got)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}
