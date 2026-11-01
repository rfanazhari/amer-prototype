package candidateinterviewusecase_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/rfanazhari/amer-prototype/application/repository/candidateinterviewport"
	"github.com/rfanazhari/amer-prototype/application/service"
	"github.com/rfanazhari/amer-prototype/application/usecase/candidateinterviewusecase"
	"github.com/rfanazhari/amer-prototype/domain/aggregate/candidateinterview"
	"github.com/rfanazhari/amer-prototype/test/mocks"
	"github.com/rfanazhari/amer-prototype/test/testdata"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

const (
	// orchConsentID is the stored acceptance the start scenarios rely on.
	orchConsentID = "11111111-1111-4111-8111-111111111111"
	// orchExecutionID is the execution lifecycle reports.
	orchExecutionID = "22222222-2222-4222-8222-222222222222"
	// orchKey is a valid idempotency key.
	orchKey = "start-1"
	// orchTab is the calling tab.
	orchTab = "tab-1"
)

// orchGraph holds the mocked ports shared by the start, takeover and cancel scenarios.
type orchGraph struct {
	current     *mocks.CurrentCandidateSession
	catalog     *mocks.NoticeCatalog
	consents    *mocks.Consents
	controllers *mocks.Controllers
	lifecycle   *mocks.ExecutionLifecycle
	clock       *mocks.Clock
	now         time.Time
}

// newOrchGraph builds strict mocks; the fixed session expires two hours after the fixture clock.
func newOrchGraph(t *testing.T) *orchGraph {
	t.Helper()
	g := &orchGraph{
		current: mocks.NewCurrentCandidateSession(t), catalog: mocks.NewNoticeCatalog(t), consents: mocks.NewConsents(t),
		controllers: mocks.NewControllers(t), lifecycle: mocks.NewExecutionLifecycle(t), clock: mocks.NewClock(t),
		now: testdata.Now(),
	}
	g.current.On("Resolve", mock.Anything).Return(testdata.CandidateSessionRecord(t), nil).Maybe()
	g.clock.On("Now").Return(func() time.Time { return g.now }).Maybe()
	return g
}

// start returns StartInterview wired to the mocks.
func (g *orchGraph) start() candidateinterviewusecase.StartInterview {
	return candidateinterviewusecase.StartInterview{
		Current: g.current, Catalog: g.catalog, Consents: g.consents, Controllers: g.controllers, Lifecycle: g.lifecycle, Clock: g.clock,
	}
}

// takeover returns Takeover wired to the mocks.
func (g *orchGraph) takeover() candidateinterviewusecase.Takeover {
	return candidateinterviewusecase.Takeover{Current: g.current, Controllers: g.controllers, Lifecycle: g.lifecycle, Clock: g.clock}
}

// cancel returns CancelInterview wired to the mocks.
func (g *orchGraph) cancel() candidateinterviewusecase.CancelInterview {
	return candidateinterviewusecase.CancelInterview{Current: g.current, Controllers: g.controllers, Lifecycle: g.lifecycle}
}

// withNotice makes the catalog return the fixture notice.
func (g *orchGraph) withNotice(t *testing.T) {
	t.Helper()
	g.catalog.On("Current", mock.Anything).Return(catalogWith(t), true).Once()
}

// withConsentVersion makes the consent store return the stored acceptance of version.
func (g *orchGraph) withConsentVersion(t *testing.T, version string) {
	t.Helper()
	snap := storedConsent(t, version)
	g.consents.On("FindByID", mock.Anything, testdata.AssignmentID, orchConsentID).Return(snap, nil).Once()
}

// withoutConsent makes the consent store report no acceptance.
func (g *orchGraph) withoutConsent() {
	g.consents.On("FindByID", mock.Anything, testdata.AssignmentID, orchConsentID).Return(candidateinterview.ConsentSnapshot{}, candidateinterviewport.ErrNotFound).Once()
}

// facts returns lifecycle facts for the fixture execution in state.
func facts(state string) service.ExecutionFacts {
	return service.ExecutionFacts{ID: orchExecutionID, AssignmentID: testdata.AssignmentID, State: state, Version: 1}
}

// controllerSnap returns a stored controller owned by tab at version.
func controllerSnap(t *testing.T, tab string, version int64) candidateinterview.ControllerSnapshot {
	t.Helper()
	id, err := candidateinterview.ParseControllerID(tab)
	require.NoError(t, err)
	return candidateinterview.ControllerSnapshot{
		AssignmentID: testdata.AssignmentID, ExecutionID: orchExecutionID, ControllerID: id, Version: version, UpdatedAt: testdata.Now(),
	}
}

// startInput returns a valid start request.
func startInput() candidateinterviewusecase.StartInterviewInput {
	return candidateinterviewusecase.StartInterviewInput{ConsentID: orchConsentID, IdempotencyKey: orchKey, ControllerID: orchTab}
}

// commitRequest is the lifecycle request the fixture start must produce.
func commitRequest() service.CommitOrFindRequest {
	return service.CommitOrFindRequest{AssignmentID: testdata.AssignmentID, IdempotencyKey: orchKey, ConsentID: orchConsentID}
}

// TestStartInterviewCommitsAfterConsentAndCoverage calls lifecycle once and claims the controller for the starting tab.
func TestStartInterviewCommitsAfterConsentAndCoverage(t *testing.T) {
	g := newOrchGraph(t)
	g.withNotice(t)
	g.withConsentVersion(t, currentVersion)
	g.lifecycle.On("CommitOrFind", mock.Anything, commitRequest()).Return(service.CommitOrFindResult{Execution: facts("CREATED"), Created: true}, nil).Once()
	g.controllers.On("Claim", mock.Anything, mock.MatchedBy(func(s candidateinterview.ControllerSnapshot) bool {
		return s.AssignmentID == testdata.AssignmentID && s.ExecutionID == orchExecutionID && s.ControllerID.String() == orchTab &&
			s.Version == 1 && s.UpdatedAt.Equal(testdata.Now())
	})).Return(nil).Once()

	got, err := g.start().Execute(context.Background(), startInput())

	require.NoError(t, err)
	assert.True(t, got.Created)
	assert.Equal(t, orchExecutionID, got.Execution.ID)
	assert.Equal(t, "CREATED", got.Execution.State)
	assert.True(t, got.Execution.IsCurrentController)
	assert.EqualValues(t, 1, got.Execution.ControllerVersion)
}

// TestStartInterviewRetryReturnsSameExecution returns the found execution and keeps the existing controller.
func TestStartInterviewRetryReturnsSameExecution(t *testing.T) {
	g := newOrchGraph(t)
	g.withNotice(t)
	g.withConsentVersion(t, currentVersion)
	g.lifecycle.On("CommitOrFind", mock.Anything, commitRequest()).Return(service.CommitOrFindResult{Execution: facts("RUNNING"), Created: false}, nil).Once()
	g.controllers.On("Claim", mock.Anything, mock.Anything).Return(candidateinterviewport.ErrAlreadyExists).Once()
	g.controllers.On("Find", mock.Anything, testdata.AssignmentID).Return(controllerSnap(t, orchTab, 3), nil).Once()

	got, err := g.start().Execute(context.Background(), startInput())

	require.NoError(t, err)
	assert.False(t, got.Created)
	assert.Equal(t, orchExecutionID, got.Execution.ID)
	assert.Equal(t, "RUNNING", got.Execution.State)
	assert.True(t, got.Execution.IsCurrentController)
	assert.EqualValues(t, 3, got.Execution.ControllerVersion)
}

// TestStartInterviewRetryFromOtherTabIsNotController reports the existing controller as another tab.
func TestStartInterviewRetryFromOtherTabIsNotController(t *testing.T) {
	g := newOrchGraph(t)
	g.withNotice(t)
	g.withConsentVersion(t, currentVersion)
	g.lifecycle.On("CommitOrFind", mock.Anything, commitRequest()).Return(service.CommitOrFindResult{Execution: facts("CREATED")}, nil).Once()
	g.controllers.On("Claim", mock.Anything, mock.Anything).Return(candidateinterviewport.ErrAlreadyExists).Once()
	g.controllers.On("Find", mock.Anything, testdata.AssignmentID).Return(controllerSnap(t, "tab-2", 1), nil).Once()

	got, err := g.start().Execute(context.Background(), startInput())

	require.NoError(t, err)
	assert.False(t, got.Execution.IsCurrentController)
}

// TestStartInterviewDifferentConsentConflicts surfaces the lifecycle conflict for a reused key with another consent id.
func TestStartInterviewDifferentConsentConflicts(t *testing.T) {
	g := newOrchGraph(t)
	g.withNotice(t)
	g.withConsentVersion(t, currentVersion)
	g.lifecycle.On("CommitOrFind", mock.Anything, commitRequest()).Return(service.CommitOrFindResult{}, service.ErrInterviewConflict).Once()

	got, err := g.start().Execute(context.Background(), startInput())

	assert.Nil(t, got)
	assert.ErrorIs(t, err, service.ErrInterviewConflict)
}

// TestStartInterviewWithoutCurrentConsentNeverCallsLifecycle returns before the port for a missing or retired acceptance.
func TestStartInterviewWithoutCurrentConsentNeverCallsLifecycle(t *testing.T) {
	t.Run("no stored acceptance", func(t *testing.T) {
		g := newOrchGraph(t)
		g.withNotice(t)
		g.withoutConsent()
		got, err := g.start().Execute(context.Background(), startInput())
		assert.Nil(t, got)
		assert.ErrorIs(t, err, candidateinterviewusecase.ErrConsentRequired)
	})
	t.Run("acceptance of a retired version", func(t *testing.T) {
		g := newOrchGraph(t)
		g.withNotice(t)
		g.withConsentVersion(t, "notice-250101")
		got, err := g.start().Execute(context.Background(), startInput())
		assert.Nil(t, got)
		assert.ErrorIs(t, err, candidateinterviewusecase.ErrConsentRequired)
	})
}

// TestStartInterviewCoverageBoundaryNeverCallsLifecycle refuses at equality and allows one millisecond beyond.
func TestStartInterviewCoverageBoundaryNeverCallsLifecycle(t *testing.T) {
	expires := testdata.Now().Add(2 * time.Hour)
	t.Run("session ends exactly at the deadline", func(t *testing.T) {
		g := newOrchGraph(t)
		g.now = expires.Add(-candidateinterview.InterviewBudget)
		g.withNotice(t)
		g.withConsentVersion(t, currentVersion)
		got, err := g.start().Execute(context.Background(), startInput())
		assert.Nil(t, got)
		assert.ErrorIs(t, err, service.ErrInterviewCoverage)
	})
	t.Run("session outlasts the deadline by one millisecond", func(t *testing.T) {
		g := newOrchGraph(t)
		g.now = expires.Add(-candidateinterview.InterviewBudget - time.Millisecond)
		g.withNotice(t)
		g.withConsentVersion(t, currentVersion)
		g.lifecycle.On("CommitOrFind", mock.Anything, commitRequest()).Return(service.CommitOrFindResult{Execution: facts("CREATED"), Created: true}, nil).Once()
		g.controllers.On("Claim", mock.Anything, mock.Anything).Return(nil).Once()
		got, err := g.start().Execute(context.Background(), startInput())
		require.NoError(t, err)
		assert.Equal(t, orchExecutionID, got.Execution.ID)
	})
}

// TestStartInterviewRejectsInputBeforeAnyPort validates the idempotency key, consent id and tab before reading anything.
func TestStartInterviewRejectsInputBeforeAnyPort(t *testing.T) {
	tests := map[string]struct {
		mutate func(*candidateinterviewusecase.StartInterviewInput)
		want   error
	}{
		"blank key":      {func(in *candidateinterviewusecase.StartInterviewInput) { in.IdempotencyKey = "" }, candidateinterviewusecase.ErrInvalidIdempotencyKey},
		"leading symbol": {func(in *candidateinterviewusecase.StartInterviewInput) { in.IdempotencyKey = "-key" }, candidateinterviewusecase.ErrInvalidIdempotencyKey},
		"space in key":   {func(in *candidateinterviewusecase.StartInterviewInput) { in.IdempotencyKey = "a b" }, candidateinterviewusecase.ErrInvalidIdempotencyKey},
		"key too long":   {func(in *candidateinterviewusecase.StartInterviewInput) { in.IdempotencyKey = strings.Repeat("a", 129) }, candidateinterviewusecase.ErrInvalidIdempotencyKey},
		"bad tab":        {func(in *candidateinterviewusecase.StartInterviewInput) { in.ControllerID = " tab" }, candidateinterview.ErrInvalidControllerID},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			g := newOrchGraph(t)
			in := startInput()
			tc.mutate(&in)
			got, err := g.start().Execute(context.Background(), in)
			assert.Nil(t, got)
			assert.ErrorIs(t, err, tc.want)
		})
	}
}

// TestStartInterviewFailsClosedWithoutNoticeOrLifecycle reports a dependency failure and writes no consent.
func TestStartInterviewFailsClosedWithoutNoticeOrLifecycle(t *testing.T) {
	t.Run("empty catalog", func(t *testing.T) {
		g := newOrchGraph(t)
		g.catalog.On("Current", mock.Anything).Return(service.Notice{}, false).Once()
		got, err := g.start().Execute(context.Background(), startInput())
		assert.Nil(t, got)
		assert.ErrorIs(t, err, service.ErrInterviewDependencyUnavailable)
	})
	t.Run("unbound lifecycle port", func(t *testing.T) {
		g := newOrchGraph(t)
		g.withNotice(t)
		g.withConsentVersion(t, currentVersion)
		g.lifecycle.On("CommitOrFind", mock.Anything, commitRequest()).Return(service.CommitOrFindResult{}, service.ErrInterviewDependencyUnavailable).Once()
		got, err := g.start().Execute(context.Background(), startInput())
		assert.Nil(t, got)
		assert.ErrorIs(t, err, service.ErrInterviewDependencyUnavailable)
		g.consents.AssertNotCalled(t, "Insert", mock.Anything, mock.Anything)
	})
	t.Run("unbound session", func(t *testing.T) {
		g := newOrchGraph(t)
		g.current.ExpectedCalls = nil
		g.current.On("Resolve", mock.Anything).Return(testdata.CandidateSessionRecord(t), service.ErrCandidateSessionUnbound).Once()
		got, err := g.start().Execute(context.Background(), startInput())
		assert.Nil(t, got)
		assert.ErrorIs(t, err, service.ErrCandidateSessionUnbound)
	})
}

// TestTakeoverReplacesControllerWhenVersionMatches stores the calling tab and reports the new version.
func TestTakeoverReplacesControllerWhenVersionMatches(t *testing.T) {
	g := newOrchGraph(t)
	g.lifecycle.On("Find", mock.Anything, testdata.AssignmentID).Return(facts("RUNNING"), nil).Once()
	g.controllers.On("Update", mock.Anything, testdata.AssignmentID, mock.Anything).Return(func(_ context.Context, _ string, fn func(*candidateinterview.InterviewController) error) error {
		c, err := candidateinterview.RestoreController(controllerSnap(t, "tab-1", 4))
		require.NoError(t, err)
		require.NoError(t, fn(c))
		after := c.Snapshot()
		assert.Equal(t, "tab-2", after.ControllerID.String())
		assert.EqualValues(t, 5, after.Version)
		assert.True(t, after.UpdatedAt.Equal(testdata.Now()))
		return nil
	}).Once()

	got, err := g.takeover().Execute(context.Background(), candidateinterviewusecase.TakeoverInput{ExecutionID: orchExecutionID, ControllerID: "tab-2", ExpectedVersion: 4})

	require.NoError(t, err)
	assert.True(t, got.IsCurrentController)
	assert.EqualValues(t, 5, got.ControllerVersion)
}

// TestTakeoverRefusals maps a stale version, a terminal execution and a foreign execution id.
func TestTakeoverRefusals(t *testing.T) {
	update := func(g *orchGraph, expected int64) {
		g.controllers.On("Update", mock.Anything, testdata.AssignmentID, mock.Anything).Return(func(_ context.Context, _ string, fn func(*candidateinterview.InterviewController) error) error {
			c, err := candidateinterview.RestoreController(controllerSnap(t, "tab-1", 4))
			require.NoError(t, err)
			return fn(c)
		}).Once()
		_ = expected
	}
	t.Run("stale version is a conflict", func(t *testing.T) {
		g := newOrchGraph(t)
		g.lifecycle.On("Find", mock.Anything, testdata.AssignmentID).Return(facts("RUNNING"), nil).Once()
		update(g, 3)
		got, err := g.takeover().Execute(context.Background(), candidateinterviewusecase.TakeoverInput{ExecutionID: orchExecutionID, ControllerID: "tab-2", ExpectedVersion: 3})
		assert.Nil(t, got)
		assert.ErrorIs(t, err, service.ErrInterviewConflict)
	})
	t.Run("terminal execution is refused", func(t *testing.T) {
		g := newOrchGraph(t)
		g.lifecycle.On("Find", mock.Anything, testdata.AssignmentID).Return(facts("COMPLETED"), nil).Once()
		update(g, 4)
		got, err := g.takeover().Execute(context.Background(), candidateinterviewusecase.TakeoverInput{ExecutionID: orchExecutionID, ControllerID: "tab-2", ExpectedVersion: 4})
		assert.Nil(t, got)
		assert.ErrorIs(t, err, candidateinterview.ErrExecutionTerminal)
	})
	t.Run("execution outside the assignment is not found", func(t *testing.T) {
		g := newOrchGraph(t)
		g.lifecycle.On("Find", mock.Anything, testdata.AssignmentID).Return(facts("RUNNING"), nil).Once()
		got, err := g.takeover().Execute(context.Background(), candidateinterviewusecase.TakeoverInput{ExecutionID: "33333333-3333-4333-8333-333333333333", ControllerID: "tab-2", ExpectedVersion: 4})
		assert.Nil(t, got)
		assert.ErrorIs(t, err, service.ErrInterviewNotFound)
	})
	t.Run("invalid tab id is rejected before any port", func(t *testing.T) {
		g := newOrchGraph(t)
		got, err := g.takeover().Execute(context.Background(), candidateinterviewusecase.TakeoverInput{ExecutionID: orchExecutionID, ControllerID: "", ExpectedVersion: 4})
		assert.Nil(t, got)
		assert.ErrorIs(t, err, candidateinterview.ErrInvalidControllerID)
	})
}

// TestCancelInterviewByControllerAsksForEarlyStop sends candidate_early_stop and returns the CANCELLED execution.
func TestCancelInterviewByControllerAsksForEarlyStop(t *testing.T) {
	g := newOrchGraph(t)
	g.lifecycle.On("Find", mock.Anything, testdata.AssignmentID).Return(facts("RUNNING"), nil).Once()
	g.controllers.On("Find", mock.Anything, testdata.AssignmentID).Return(controllerSnap(t, orchTab, 2), nil).Once()
	cancelled := facts("CANCELLED")
	cancelled.Version = 2
	g.lifecycle.On("Cancel", mock.Anything, service.CancelRequest{ExecutionID: orchExecutionID, ExpectedVersion: 1, Reason: service.CancelReasonCandidateEarlyStop}).Return(cancelled, nil).Once()

	got, err := g.cancel().Execute(context.Background(), candidateinterviewusecase.CancelInterviewInput{ExecutionID: orchExecutionID, ControllerID: orchTab, ExpectedVersion: 1})

	require.NoError(t, err)
	assert.Equal(t, "CANCELLED", got.State)
	assert.EqualValues(t, 2, got.Version)
	assert.True(t, got.IsCurrentController)
	assert.EqualValues(t, 2, got.ControllerVersion)
}

// TestCancelInterviewRefusalsNeverCallLifecycleCancel keeps the port untouched for a non-controller or finished execution.
func TestCancelInterviewRefusalsNeverCallLifecycleCancel(t *testing.T) {
	t.Run("non-controller", func(t *testing.T) {
		g := newOrchGraph(t)
		g.lifecycle.On("Find", mock.Anything, testdata.AssignmentID).Return(facts("RUNNING"), nil).Once()
		g.controllers.On("Find", mock.Anything, testdata.AssignmentID).Return(controllerSnap(t, "tab-2", 2), nil).Once()
		got, err := g.cancel().Execute(context.Background(), candidateinterviewusecase.CancelInterviewInput{ExecutionID: orchExecutionID, ControllerID: orchTab, ExpectedVersion: 1})
		assert.Nil(t, got)
		assert.ErrorIs(t, err, candidateinterviewusecase.ErrNotController)
	})
	t.Run("terminal execution is not relabelled", func(t *testing.T) {
		g := newOrchGraph(t)
		g.lifecycle.On("Find", mock.Anything, testdata.AssignmentID).Return(facts("COMPLETED"), nil).Once()
		got, err := g.cancel().Execute(context.Background(), candidateinterviewusecase.CancelInterviewInput{ExecutionID: orchExecutionID, ControllerID: orchTab, ExpectedVersion: 1})
		assert.Nil(t, got)
		assert.ErrorIs(t, err, candidateinterview.ErrExecutionTerminal)
	})
	t.Run("execution outside the assignment is not found", func(t *testing.T) {
		g := newOrchGraph(t)
		g.lifecycle.On("Find", mock.Anything, testdata.AssignmentID).Return(facts("RUNNING"), nil).Once()
		got, err := g.cancel().Execute(context.Background(), candidateinterviewusecase.CancelInterviewInput{ExecutionID: "33333333-3333-4333-8333-333333333333", ControllerID: orchTab, ExpectedVersion: 1})
		assert.Nil(t, got)
		assert.ErrorIs(t, err, service.ErrInterviewNotFound)
	})
	t.Run("unbound lifecycle port", func(t *testing.T) {
		g := newOrchGraph(t)
		g.lifecycle.On("Find", mock.Anything, testdata.AssignmentID).Return(service.ExecutionFacts{}, service.ErrInterviewDependencyUnavailable).Once()
		got, err := g.cancel().Execute(context.Background(), candidateinterviewusecase.CancelInterviewInput{ExecutionID: orchExecutionID, ControllerID: orchTab, ExpectedVersion: 1})
		assert.Nil(t, got)
		assert.ErrorIs(t, err, service.ErrInterviewDependencyUnavailable)
	})
	t.Run("stale version is a lifecycle conflict", func(t *testing.T) {
		g := newOrchGraph(t)
		g.lifecycle.On("Find", mock.Anything, testdata.AssignmentID).Return(facts("RUNNING"), nil).Once()
		g.controllers.On("Find", mock.Anything, testdata.AssignmentID).Return(controllerSnap(t, orchTab, 2), nil).Once()
		g.lifecycle.On("Cancel", mock.Anything, mock.Anything).Return(service.ExecutionFacts{}, service.ErrInterviewConflict).Once()
		got, err := g.cancel().Execute(context.Background(), candidateinterviewusecase.CancelInterviewInput{ExecutionID: orchExecutionID, ControllerID: orchTab, ExpectedVersion: 0})
		assert.Nil(t, got)
		assert.ErrorIs(t, err, service.ErrInterviewConflict)
	})
}
