package interviewreport_test

import (
	"testing"
	"time"

	"github.com/rfanazhari/amer-prototype/domain/aggregate/interviewreport"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fixtureNow returns a reproducible UTC millisecond-precision instant.
func fixtureNow() time.Time { return time.Date(2026, 11, 3, 9, 0, 0, 123000000, time.UTC) }

// fixtureReport creates a report with no generation or fails the test.
func fixtureReport(t *testing.T) *interviewreport.Report {
	t.Helper()
	r, err := interviewreport.NewReport("report-1", "exec-1", "assign-1", fixtureNow())
	require.NoError(t, err)
	require.NotNil(t, r)
	return r
}

// fixtureStatements returns one valid observation and one gap.
func fixtureStatements(t *testing.T) []interviewreport.Statement {
	t.Helper()
	obs, err := interviewreport.NewStatement(interviewreport.StatementInput{Kind: interviewreport.KindObservation, Body: "Answered.", PresentsAsAnswer: true, Citation: fixtureCitation()})
	require.NoError(t, err)
	gap, err := interviewreport.NewStatement(interviewreport.StatementInput{Kind: interviewreport.KindGap, Body: "Not covered."})
	require.NoError(t, err)
	return []interviewreport.Statement{obs, gap}
}

// TestBudgetConstants pins the accepted attempt budget.
func TestBudgetConstants(t *testing.T) {
	assert.Equal(t, 2, interviewreport.MaxAttempts)
	assert.Equal(t, 60*time.Second, interviewreport.AttemptTimeout)
	assert.Equal(t, 5*time.Second, interviewreport.RetryDelay)
	assert.Equal(t, 5*time.Minute, interviewreport.JobWindow)
	assert.Equal(t, 30*time.Minute, interviewreport.EvidenceWait)
	assert.Equal(t, 30*time.Second, interviewreport.EvidenceRecheck)
}

// TestNewReport covers construction validation.
func TestNewReport(t *testing.T) {
	// newCase is one row in the NewReport table-driven test.
	type newCase struct {
		name         string
		id           string
		executionID  string
		assignmentID string
		now          time.Time
		wantErr      bool
	}
	cases := []newCase{
		{name: "valid", id: "r", executionID: "e", assignmentID: "a", now: fixtureNow()},
		{name: "empty id", executionID: "e", assignmentID: "a", now: fixtureNow(), wantErr: true},
		{name: "empty execution", id: "r", assignmentID: "a", now: fixtureNow(), wantErr: true},
		{name: "empty assignment", id: "r", executionID: "e", now: fixtureNow(), wantErr: true},
		{name: "zero time", id: "r", executionID: "e", assignmentID: "a", wantErr: true},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			r, err := interviewreport.NewReport(tt.id, tt.executionID, tt.assignmentID, tt.now)
			if tt.wantErr {
				assert.ErrorIs(t, err, interviewreport.ErrInvalidReport)
				return
			}
			require.NoError(t, err)
			s := r.Snapshot()
			assert.Equal(t, tt.id, s.ID)
			assert.Equal(t, tt.executionID, s.ExecutionID)
			assert.Equal(t, tt.assignmentID, s.AssignmentID)
			assert.Equal(t, tt.now, s.CreatedAt)
			assert.Empty(t, s.Generations)
		})
	}
}

// TestStartGeneration covers the one-active-generation rule and numbering.
func TestStartGeneration(t *testing.T) {
	// startCase is one row in the StartGeneration table-driven test.
	type startCase struct {
		name       string
		prepare    func(t *testing.T, r *interviewreport.Report)
		wantNumber int
		wantErr    error
	}
	cases := []startCase{
		{
			name:       "first generation",
			prepare:    func(t *testing.T, r *interviewreport.Report) {},
			wantNumber: 1,
		},
		{
			name: "refused while pending",
			prepare: func(t *testing.T, r *interviewreport.Report) {
				_, err := r.StartGeneration("g1", fixtureNow())
				require.NoError(t, err)
			},
			wantErr: interviewreport.ErrGenerationActive,
		},
		{
			name: "refused while running",
			prepare: func(t *testing.T, r *interviewreport.Report) {
				_, err := r.StartGeneration("g1", fixtureNow())
				require.NoError(t, err)
				_, err = r.Run("g1", fixtureNow())
				require.NoError(t, err)
			},
			wantErr: interviewreport.ErrGenerationActive,
		},
		{
			name: "new generation after failure",
			prepare: func(t *testing.T, r *interviewreport.Report) {
				_, err := r.StartGeneration("g1", fixtureNow())
				require.NoError(t, err)
				_, err = r.Fail("g1", "evidence_unavailable", fixtureNow())
				require.NoError(t, err)
			},
			wantNumber: 2,
		},
		{
			name: "new generation after success",
			prepare: func(t *testing.T, r *interviewreport.Report) {
				_, err := r.StartGeneration("g1", fixtureNow())
				require.NoError(t, err)
				_, err = r.Run("g1", fixtureNow())
				require.NoError(t, err)
				_, err = r.Succeed("g1", fixtureStatements(t), fixtureNow())
				require.NoError(t, err)
			},
			wantNumber: 2,
		},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			r := fixtureReport(t)
			tt.prepare(t, r)
			before := r.Snapshot()

			g, err := r.StartGeneration("g-next", fixtureNow().Add(time.Minute))

			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				assert.Equal(t, before, r.Snapshot())
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantNumber, g.Number)
			assert.Equal(t, interviewreport.StatePending, g.State)
			assert.Equal(t, "g-next", g.ID)
			assert.Len(t, r.Snapshot().Generations, tt.wantNumber)
		})
	}
}

// TestStartGenerationRejectsInvalidInput covers identity and time validation.
func TestStartGenerationRejectsInvalidInput(t *testing.T) {
	r := fixtureReport(t)

	_, err := r.StartGeneration("", fixtureNow())
	assert.ErrorIs(t, err, interviewreport.ErrInvalidReport)
	_, err = r.StartGeneration("g1", time.Time{})
	assert.ErrorIs(t, err, interviewreport.ErrInvalidReport)
	assert.Empty(t, r.Snapshot().Generations)
}

// TestGenerationTransitions covers run, succeed, and fail transitions.
func TestGenerationTransitions(t *testing.T) {
	// transitionCase is one row in the generation transition table-driven test.
	type transitionCase struct {
		name      string
		prepare   func(t *testing.T, r *interviewreport.Report)
		act       func(r *interviewreport.Report) (interviewreport.Generation, error)
		wantState interviewreport.State
		wantReson string
		wantErr   error
	}
	later := fixtureNow().Add(time.Second)
	cases := []transitionCase{
		{
			name:      "run pending",
			prepare:   startPending,
			act:       func(r *interviewreport.Report) (interviewreport.Generation, error) { return r.Run("g1", later) },
			wantState: interviewreport.StateRunning,
		},
		{
			name:    "run unknown generation",
			prepare: startPending,
			act:     func(r *interviewreport.Report) (interviewreport.Generation, error) { return r.Run("nope", later) },
			wantErr: interviewreport.ErrGenerationNotFound,
		},
		{
			name:    "run twice",
			prepare: startRunning,
			act:     func(r *interviewreport.Report) (interviewreport.Generation, error) { return r.Run("g1", later) },
			wantErr: interviewreport.ErrInvalidTransition,
		},
		{
			name:    "fail pending",
			prepare: startPending,
			act: func(r *interviewreport.Report) (interviewreport.Generation, error) {
				return r.Fail("g1", "evidence_unavailable", later)
			},
			wantState: interviewreport.StateFailed,
			wantReson: "evidence_unavailable",
		},
		{
			name:    "fail running",
			prepare: startRunning,
			act: func(r *interviewreport.Report) (interviewreport.Generation, error) {
				return r.Fail("g1", "attempts_exhausted", later)
			},
			wantState: interviewreport.StateFailed,
			wantReson: "attempts_exhausted",
		},
		{
			name:    "fail without reason",
			prepare: startRunning,
			act:     func(r *interviewreport.Report) (interviewreport.Generation, error) { return r.Fail("g1", "", later) },
			wantErr: interviewreport.ErrInvalidReport,
		},
		{
			name:    "fail twice",
			prepare: startFailed,
			act: func(r *interviewreport.Report) (interviewreport.Generation, error) {
				return r.Fail("g1", "again", later)
			},
			wantErr: interviewreport.ErrInvalidTransition,
		},
		{
			name:    "succeed running",
			prepare: startRunning,
			act: func(r *interviewreport.Report) (interviewreport.Generation, error) {
				return r.Succeed("g1", fixtureStatements(t), later)
			},
			wantState: interviewreport.StateSucceeded,
		},
		{
			name:    "succeed pending",
			prepare: startPending,
			act: func(r *interviewreport.Report) (interviewreport.Generation, error) {
				return r.Succeed("g1", fixtureStatements(t), later)
			},
			wantErr: interviewreport.ErrInvalidTransition,
		},
		{
			name:    "succeed after failure",
			prepare: startFailed,
			act: func(r *interviewreport.Report) (interviewreport.Generation, error) {
				return r.Succeed("g1", fixtureStatements(t), later)
			},
			wantErr: interviewreport.ErrInvalidTransition,
		},
		{
			name:    "succeed with zero value statement",
			prepare: startRunning,
			act: func(r *interviewreport.Report) (interviewreport.Generation, error) {
				return r.Succeed("g1", []interviewreport.Statement{{}}, later)
			},
			wantErr: interviewreport.ErrInvalidStatement,
		},
		{
			name:    "time before creation",
			prepare: startRunning,
			act: func(r *interviewreport.Report) (interviewreport.Generation, error) {
				return r.Fail("g1", "late", fixtureNow().Add(-time.Second))
			},
			wantErr: interviewreport.ErrInvalidReport,
		},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			r := fixtureReport(t)
			tt.prepare(t, r)
			before := r.Snapshot()

			g, err := tt.act(r)

			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				assert.Equal(t, before, r.Snapshot())
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantState, g.State)
			assert.Equal(t, tt.wantReson, g.FailureReason)
			assert.Equal(t, later, g.UpdatedAt)
			stored := r.Snapshot().Generations[0]
			assert.Equal(t, g, stored)
		})
	}
}

// TestSucceedStoresStatementsInOrder verifies accepted output keeps order and is detached from the caller.
func TestSucceedStoresStatementsInOrder(t *testing.T) {
	r := fixtureReport(t)
	startRunning(t, r)
	statements := fixtureStatements(t)

	g, err := r.Succeed("g1", statements, fixtureNow())
	require.NoError(t, err)
	statements[0] = statements[1]
	g.Statements[1] = statements[0]

	stored := r.Snapshot().Generations[0].Statements
	require.Len(t, stored, 2)
	assert.Equal(t, interviewreport.KindObservation, stored[0].Kind())
	assert.Equal(t, interviewreport.KindGap, stored[1].Kind())
}

// TestFailureKeepsPreviousSuccessfulAccount verifies a failed regeneration does not erase earlier statements.
func TestFailureKeepsPreviousSuccessfulAccount(t *testing.T) {
	r := fixtureReport(t)
	startRunning(t, r)
	_, err := r.Succeed("g1", fixtureStatements(t), fixtureNow())
	require.NoError(t, err)
	_, err = r.StartGeneration("g2", fixtureNow())
	require.NoError(t, err)

	_, err = r.Fail("g2", "provider_failed", fixtureNow())
	require.NoError(t, err)

	gens := r.Snapshot().Generations
	require.Len(t, gens, 2)
	assert.Equal(t, interviewreport.StateSucceeded, gens[0].State)
	assert.Len(t, gens[0].Statements, 2)
	assert.Equal(t, interviewreport.StateFailed, gens[1].State)
	assert.Empty(t, gens[1].Statements)
}

// startPending adds a pending generation g1.
func startPending(t *testing.T, r *interviewreport.Report) {
	t.Helper()
	_, err := r.StartGeneration("g1", fixtureNow())
	require.NoError(t, err)
}

// startRunning adds a running generation g1.
func startRunning(t *testing.T, r *interviewreport.Report) {
	t.Helper()
	startPending(t, r)
	_, err := r.Run("g1", fixtureNow())
	require.NoError(t, err)
}

// startFailed adds a failed generation g1.
func startFailed(t *testing.T, r *interviewreport.Report) {
	t.Helper()
	startPending(t, r)
	_, err := r.Fail("g1", "provider_failed", fixtureNow())
	require.NoError(t, err)
}
