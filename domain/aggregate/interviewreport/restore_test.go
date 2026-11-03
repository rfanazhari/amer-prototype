package interviewreport_test

import (
	"testing"
	"time"

	"github.com/rfanazhari/amer-prototype/domain/aggregate/interviewreport"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fixtureHistory returns a stored report with a failed, a succeeded, and a pending generation.
func fixtureHistory(t *testing.T) interviewreport.Snapshot {
	t.Helper()
	t0 := fixtureNow()
	return interviewreport.Snapshot{
		ID: "report-1", ExecutionID: "exec-1", AssignmentID: "assign-1", CreatedAt: t0,
		Generations: []interviewreport.Generation{
			{ID: "g1", Number: 1, State: interviewreport.StateFailed, FailureReason: "provider_failed", CreatedAt: t0, UpdatedAt: t0.Add(time.Minute)},
			{ID: "g2", Number: 2, State: interviewreport.StateSucceeded, CreatedAt: t0.Add(time.Hour), UpdatedAt: t0.Add(time.Hour), Statements: fixtureStatements(t)},
			{ID: "g3", Number: 3, State: interviewreport.StatePending, CreatedAt: t0.Add(2 * time.Hour), UpdatedAt: t0.Add(2 * time.Hour)},
		},
	}
}

// TestRestoreReport covers acceptance of a consistent history and rejection of corrupt rows.
func TestRestoreReport(t *testing.T) {
	// restoreCase is one row in the RestoreReport table-driven test.
	type restoreCase struct {
		name    string
		mutate  func(s *interviewreport.Snapshot)
		wantErr bool
	}
	cases := []restoreCase{
		{name: "pending failed and succeeded history", mutate: func(s *interviewreport.Snapshot) {}},
		{name: "no generation", mutate: func(s *interviewreport.Snapshot) { s.Generations = nil }},
		{name: "empty report id", mutate: func(s *interviewreport.Snapshot) { s.ID = "" }, wantErr: true},
		{name: "empty execution id", mutate: func(s *interviewreport.Snapshot) { s.ExecutionID = "" }, wantErr: true},
		{name: "unknown state", mutate: func(s *interviewreport.Snapshot) { s.Generations[2].State = "queued" }, wantErr: true},
		{
			name:    "two active generations",
			mutate:  func(s *interviewreport.Snapshot) { s.Generations[0] = pendingAs(s.Generations[0]) },
			wantErr: true,
		},
		{name: "number gap", mutate: func(s *interviewreport.Snapshot) { s.Generations[2].Number = 4 }, wantErr: true},
		{name: "failed without reason", mutate: func(s *interviewreport.Snapshot) { s.Generations[0].FailureReason = "" }, wantErr: true},
		{name: "reason on succeeded", mutate: func(s *interviewreport.Snapshot) { s.Generations[1].FailureReason = "x" }, wantErr: true},
		{name: "statements on failed", mutate: func(s *interviewreport.Snapshot) { s.Generations[0].Statements = s.Generations[1].Statements }, wantErr: true},
		{name: "statements on pending", mutate: func(s *interviewreport.Snapshot) { s.Generations[2].Statements = s.Generations[1].Statements }, wantErr: true},
		{name: "updated before created", mutate: func(s *interviewreport.Snapshot) {
			s.Generations[2].UpdatedAt = s.Generations[2].CreatedAt.Add(-time.Second)
		}, wantErr: true},
		{name: "zero value statement", mutate: func(s *interviewreport.Snapshot) { s.Generations[1].Statements = []interviewreport.Statement{{}} }, wantErr: true},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			s := fixtureHistory(t)
			tt.mutate(&s)

			r, err := interviewreport.RestoreReport(s)

			if tt.wantErr {
				assert.ErrorIs(t, err, interviewreport.ErrInvalidReport)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, s, r.Snapshot())
		})
	}
}

// TestRestoredReportContinuesLifecycle verifies a restored history keeps its numbering and active rule.
func TestRestoredReportContinuesLifecycle(t *testing.T) {
	r, err := interviewreport.RestoreReport(fixtureHistory(t))
	require.NoError(t, err)

	_, err = r.StartGeneration("g4", fixtureNow().Add(3*time.Hour))
	assert.ErrorIs(t, err, interviewreport.ErrGenerationActive)

	_, err = r.Fail("g3", "evidence_unavailable", fixtureNow().Add(3*time.Hour))
	require.NoError(t, err)
	g, err := r.StartGeneration("g4", fixtureNow().Add(4*time.Hour))
	require.NoError(t, err)
	assert.Equal(t, 4, g.Number)
}

// TestRestoreStatement covers shape checks on stored statements without the writer wording guard.
func TestRestoreStatement(t *testing.T) {
	// restoreStatementCase is one row in the RestoreStatement table-driven test.
	type restoreStatementCase struct {
		name    string
		input   interviewreport.StatementInput
		wantErr error
	}
	cases := []restoreStatementCase{
		{
			name:  "stored text with a verdict word is not rejected",
			input: interviewreport.StatementInput{Kind: interviewreport.KindUncertainty, Body: "Earlier writer said hire."},
		},
		{
			name:    "answer without citation",
			input:   interviewreport.StatementInput{Kind: interviewreport.KindObservation, Body: "Answered.", PresentsAsAnswer: true},
			wantErr: interviewreport.ErrCitationRequired,
		},
		{
			name:    "gap with citation",
			input:   interviewreport.StatementInput{Kind: interviewreport.KindGap, Body: "Missing.", Citation: fixtureCitation()},
			wantErr: interviewreport.ErrGapShape,
		},
		{
			name:    "unknown kind",
			input:   interviewreport.StatementInput{Kind: interviewreport.Kind("rank"), Body: "First."},
			wantErr: interviewreport.ErrInvalidStatement,
		},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			s, err := interviewreport.RestoreStatement(tt.input)
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.input.Body, s.Body())
		})
	}
}

// pendingAs returns g relabelled as an active pending generation with no failure reason.
func pendingAs(g interviewreport.Generation) interviewreport.Generation {
	g.State = interviewreport.StatePending
	g.FailureReason = ""
	return g
}
