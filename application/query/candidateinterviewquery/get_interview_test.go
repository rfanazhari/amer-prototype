package candidateinterviewquery_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/rfanazhari/amer-prototype/application/query/candidateinterviewquery"
	"github.com/rfanazhari/amer-prototype/application/repository/candidateinterviewport"
	"github.com/rfanazhari/amer-prototype/application/service"
	"github.com/rfanazhari/amer-prototype/domain/aggregate/candidateinterview"
	"github.com/rfanazhari/amer-prototype/test/mocks"
	"github.com/rfanazhari/amer-prototype/test/testdata"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// tabID is the calling tab used by the fixtures.
const tabID = "tab-1"

// readGraph holds the mocked ports of one GetInterview scenario.
type readGraph struct {
	current     *mocks.CurrentCandidateSession
	catalog     *mocks.NoticeCatalog
	consents    *mocks.Consents
	controllers *mocks.Controllers
	lifecycle   *mocks.ExecutionLifecycle
	progress    *mocks.InterviewProgress
	clock       *mocks.Clock
}

// newReadGraph builds strict mocks; the fixed session expires two hours after the fixture clock.
func newReadGraph(t *testing.T) readGraph {
	t.Helper()
	g := readGraph{
		current: mocks.NewCurrentCandidateSession(t), catalog: mocks.NewNoticeCatalog(t), consents: mocks.NewConsents(t),
		controllers: mocks.NewControllers(t), lifecycle: mocks.NewExecutionLifecycle(t), progress: mocks.NewInterviewProgress(t),
		clock: mocks.NewClock(t),
	}
	g.current.On("Resolve", mock.Anything).Return(testdata.CandidateSessionRecord(t), nil).Maybe()
	g.clock.On("Now").Return(testdata.Now()).Maybe()
	return g
}

// operation returns the query wired to the mocks.
func (g readGraph) operation() candidateinterviewquery.GetInterview {
	return candidateinterviewquery.GetInterview{
		Current: g.current, Catalog: g.catalog, Consents: g.consents, Controllers: g.controllers,
		Lifecycle: g.lifecycle, Progress: g.progress, Clock: g.clock,
	}
}

// withNotice makes the catalog return the fixture notice.
func (g readGraph) withNotice(t *testing.T) {
	t.Helper()
	v, err := candidateinterview.ParseNoticeVersion("notice-261102")
	require.NoError(t, err)
	g.catalog.On("Current", mock.Anything).Return(service.Notice{Version: v, Statement: "statement"}, true).Once()
}

// withConsent makes the consent store return an acceptance of the current version.
func (g readGraph) withConsent(t *testing.T) {
	t.Helper()
	v, err := candidateinterview.ParseNoticeVersion("notice-261102")
	require.NoError(t, err)
	g.consents.On("Find", mock.Anything, testdata.AssignmentID, v).Return(candidateinterview.ConsentSnapshot{
		ID: "11111111-1111-4111-8111-111111111111", AssignmentID: testdata.AssignmentID, NoticeVersion: v,
		CandidateSessionID: testdata.CandidateSessionID, AcceptedAt: testdata.Now().Add(time.Minute),
	}, nil).Once()
}

// withoutExecution makes lifecycle report that no execution exists.
func (g readGraph) withoutExecution() {
	g.lifecycle.On("Find", mock.Anything, testdata.AssignmentID).Return(service.ExecutionFacts{}, service.ErrInterviewNotFound).Once()
}

// withExecution makes lifecycle report an execution in the state.
func (g readGraph) withExecution(state string) {
	g.lifecycle.On("Find", mock.Anything, testdata.AssignmentID).Return(service.ExecutionFacts{
		ID: "22222222-2222-4222-8222-222222222222", AssignmentID: testdata.AssignmentID, State: state, Version: 3,
	}, nil).Once()
}

// withController makes the controller store return tab as the controller at version 2.
func (g readGraph) withController(t *testing.T, tab string) {
	t.Helper()
	id, err := candidateinterview.ParseControllerID(tab)
	require.NoError(t, err)
	g.controllers.On("Find", mock.Anything, testdata.AssignmentID).Return(candidateinterview.ControllerSnapshot{
		AssignmentID: testdata.AssignmentID, ExecutionID: "22222222-2222-4222-8222-222222222222", ControllerID: id, Version: 2, UpdatedAt: testdata.Now(),
	}, nil).Once()
}

// TestGetInterviewActionsBeforeExecution derives accept and start from consent and session coverage.
func TestGetInterviewActionsBeforeExecution(t *testing.T) {
	t.Run("empty catalog has no notice and no actions", func(t *testing.T) {
		g := newReadGraph(t)
		g.catalog.On("Current", mock.Anything).Return(service.Notice{}, false).Once()
		g.withoutExecution()
		got, err := g.operation().Execute(context.Background(), candidateinterviewquery.GetInterviewInput{ControllerID: tabID})
		require.NoError(t, err)
		assert.Nil(t, got.Notice)
		assert.Nil(t, got.Consent)
		assert.Nil(t, got.Execution)
		assert.Empty(t, got.Actions)
	})
	t.Run("not accepted offers accept", func(t *testing.T) {
		g := newReadGraph(t)
		g.withNotice(t)
		g.consents.On("Find", mock.Anything, testdata.AssignmentID, mock.Anything).Return(candidateinterview.ConsentSnapshot{}, candidateinterviewport.ErrNotFound).Once()
		g.withoutExecution()
		got, err := g.operation().Execute(context.Background(), candidateinterviewquery.GetInterviewInput{ControllerID: tabID})
		require.NoError(t, err)
		require.NotNil(t, got.Notice)
		assert.Equal(t, "notice-261102", got.Notice.Version)
		assert.Equal(t, "statement", got.Notice.Statement)
		assert.Nil(t, got.Consent)
		assert.Nil(t, got.Execution)
		assert.Equal(t, []string{"accept"}, got.Actions)
		assert.Nil(t, got.Progress)
	})
	t.Run("accepted and covered offers start", func(t *testing.T) {
		g := newReadGraph(t)
		g.withNotice(t)
		g.withConsent(t)
		g.withoutExecution()
		got, err := g.operation().Execute(context.Background(), candidateinterviewquery.GetInterviewInput{ControllerID: tabID})
		require.NoError(t, err)
		require.NotNil(t, got.Consent)
		assert.Equal(t, "notice-261102", got.Consent.NoticeVersion)
		assert.True(t, got.Consent.AcceptedAt.Equal(testdata.Now().Add(time.Minute)))
		assert.Nil(t, got.Execution)
		assert.Equal(t, []string{"start"}, got.Actions)
	})
}

// TestGetInterviewCoverageBoundary refuses start when the session ends at or before now plus the budget.
func TestGetInterviewCoverageBoundary(t *testing.T) {
	// boundaryCase is one session expiry relative to now plus the interview budget.
	type boundaryCase struct {
		name    string
		offset  time.Duration
		actions []string
	}
	tests := []boundaryCase{
		{name: "equality is not covered", offset: candidateinterview.InterviewBudget, actions: nil},
		{name: "one millisecond beyond is covered", offset: candidateinterview.InterviewBudget + time.Millisecond, actions: []string{"start"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := mocksWithoutDefaultSession(t)
			session := testdata.CandidateSessionRecord(t)
			session.ExpiresAt = testdata.Now().Add(tt.offset)
			g.current.On("Resolve", mock.Anything).Return(session, nil).Once()
			g.withNotice(t)
			g.withConsent(t)
			g.withoutExecution()
			got, err := g.operation().Execute(context.Background(), candidateinterviewquery.GetInterviewInput{ControllerID: tabID})
			require.NoError(t, err)
			assert.Equal(t, tt.actions, got.Actions)
		})
	}
}

// mocksWithoutDefaultSession builds the graph without the default session expectation.
func mocksWithoutDefaultSession(t *testing.T) readGraph {
	t.Helper()
	g := readGraph{
		current: mocks.NewCurrentCandidateSession(t), catalog: mocks.NewNoticeCatalog(t), consents: mocks.NewConsents(t),
		controllers: mocks.NewControllers(t), lifecycle: mocks.NewExecutionLifecycle(t), progress: mocks.NewInterviewProgress(t),
		clock: mocks.NewClock(t),
	}
	g.clock.On("Now").Return(testdata.Now()).Maybe()
	return g
}

// TestGetInterviewExecutionActions derives takeover, cancel and terminal actions and reads progress only for live states.
func TestGetInterviewExecutionActions(t *testing.T) {
	t.Run("controller tab may cancel and sees progress", func(t *testing.T) {
		g := newReadGraph(t)
		g.withNotice(t)
		g.withConsent(t)
		g.withExecution("RUNNING")
		g.withController(t, tabID)
		g.progress.On("Current", mock.Anything, "22222222-2222-4222-8222-222222222222").Return(&service.CandidateProgress{Phase: "screening", Connection: "connected"}, nil).Once()
		got, err := g.operation().Execute(context.Background(), candidateinterviewquery.GetInterviewInput{ControllerID: tabID})
		require.NoError(t, err)
		require.NotNil(t, got.Execution)
		assert.Equal(t, "RUNNING", got.Execution.State)
		assert.Equal(t, int64(3), got.Execution.Version)
		assert.True(t, got.Execution.IsCurrentController)
		assert.Equal(t, int64(2), got.Execution.ControllerVersion)
		assert.Equal(t, []string{"cancel"}, got.Actions)
		require.NotNil(t, got.Progress)
		assert.Equal(t, "screening", got.Progress.Phase)
		assert.Equal(t, "connected", got.Progress.Connection)
	})
	t.Run("other tab may take over", func(t *testing.T) {
		g := newReadGraph(t)
		g.withNotice(t)
		g.withConsent(t)
		g.withExecution("CREATED")
		g.withController(t, "tab-2")
		g.progress.On("Current", mock.Anything, mock.Anything).Return(nil, nil).Once()
		got, err := g.operation().Execute(context.Background(), candidateinterviewquery.GetInterviewInput{ControllerID: tabID})
		require.NoError(t, err)
		assert.False(t, got.Execution.IsCurrentController)
		assert.Equal(t, []string{"takeover"}, got.Actions)
		assert.Nil(t, got.Progress)
	})
	t.Run("progress failure leaves the rest of the view", func(t *testing.T) {
		g := newReadGraph(t)
		g.withNotice(t)
		g.withConsent(t)
		g.withExecution("RUNNING")
		g.withController(t, tabID)
		g.progress.On("Current", mock.Anything, mock.Anything).Return(nil, service.ErrInterviewDependencyUnavailable).Once()
		got, err := g.operation().Execute(context.Background(), candidateinterviewquery.GetInterviewInput{ControllerID: tabID})
		require.NoError(t, err)
		assert.Nil(t, got.Progress)
		assert.Equal(t, []string{"cancel"}, got.Actions)
	})
	t.Run("absent or invalid tab header is not the controller", func(t *testing.T) {
		g := newReadGraph(t)
		g.withNotice(t)
		g.withConsent(t)
		g.withExecution("CREATED")
		g.withController(t, tabID)
		g.progress.On("Current", mock.Anything, mock.Anything).Return(nil, nil).Once()
		got, err := g.operation().Execute(context.Background(), candidateinterviewquery.GetInterviewInput{ControllerID: ""})
		require.NoError(t, err)
		assert.False(t, got.Execution.IsCurrentController)
		assert.Equal(t, []string{"takeover"}, got.Actions)
	})
	t.Run("missing controller row is not current", func(t *testing.T) {
		g := newReadGraph(t)
		g.withNotice(t)
		g.withConsent(t)
		g.withExecution("CREATED")
		g.controllers.On("Find", mock.Anything, testdata.AssignmentID).Return(candidateinterview.ControllerSnapshot{}, candidateinterviewport.ErrNotFound).Once()
		g.progress.On("Current", mock.Anything, mock.Anything).Return(nil, nil).Once()
		got, err := g.operation().Execute(context.Background(), candidateinterviewquery.GetInterviewInput{ControllerID: tabID})
		require.NoError(t, err)
		assert.False(t, got.Execution.IsCurrentController)
		assert.Equal(t, int64(0), got.Execution.ControllerVersion)
	})
	for _, state := range []string{"COMPLETED", "FAILED", "CANCELLED"} {
		t.Run("terminal "+state+" has no actions and no progress read", func(t *testing.T) {
			g := newReadGraph(t)
			g.withNotice(t)
			g.withConsent(t)
			g.withExecution(state)
			g.withController(t, tabID)
			got, err := g.operation().Execute(context.Background(), candidateinterviewquery.GetInterviewInput{ControllerID: tabID})
			require.NoError(t, err)
			assert.Equal(t, state, got.Execution.State)
			assert.Empty(t, got.Actions)
			assert.Nil(t, got.Progress)
		})
	}
}

// TestGetInterviewDependencyAndSessionFailures covers port absence, storage failure and an unbound session.
func TestGetInterviewDependencyAndSessionFailures(t *testing.T) {
	t.Run("unbound lifecycle adapter renders no execution", func(t *testing.T) {
		g := newReadGraph(t)
		g.withNotice(t)
		g.withConsent(t)
		g.lifecycle.On("Find", mock.Anything, testdata.AssignmentID).Return(service.ExecutionFacts{}, service.ErrInterviewDependencyUnavailable).Once()
		got, err := g.operation().Execute(context.Background(), candidateinterviewquery.GetInterviewInput{ControllerID: tabID})
		require.NoError(t, err)
		assert.Nil(t, got.Execution)
		assert.Equal(t, []string{"start"}, got.Actions)
	})
	t.Run("lifecycle storage failure is returned", func(t *testing.T) {
		g := newReadGraph(t)
		boom := errors.New("boom")
		g.withNotice(t)
		g.withConsent(t)
		g.lifecycle.On("Find", mock.Anything, testdata.AssignmentID).Return(service.ExecutionFacts{}, boom).Once()
		got, err := g.operation().Execute(context.Background(), candidateinterviewquery.GetInterviewInput{ControllerID: tabID})
		require.ErrorIs(t, err, boom)
		assert.Nil(t, got)
	})
	t.Run("consent storage failure is returned", func(t *testing.T) {
		g := newReadGraph(t)
		boom := errors.New("boom")
		g.withNotice(t)
		g.consents.On("Find", mock.Anything, testdata.AssignmentID, mock.Anything).Return(candidateinterview.ConsentSnapshot{}, boom).Once()
		got, err := g.operation().Execute(context.Background(), candidateinterviewquery.GetInterviewInput{ControllerID: tabID})
		require.ErrorIs(t, err, boom)
		assert.Nil(t, got)
	})
	t.Run("unbound session stops the read", func(t *testing.T) {
		g := mocksWithoutDefaultSession(t)
		g.current.On("Resolve", mock.Anything).Return(testdata.CandidateSessionRecord(t), service.ErrCandidateSessionUnbound).Once()
		got, err := g.operation().Execute(context.Background(), candidateinterviewquery.GetInterviewInput{ControllerID: tabID})
		require.ErrorIs(t, err, service.ErrCandidateSessionUnbound)
		assert.Nil(t, got)
	})
}
