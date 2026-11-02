package integration_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/rfanazhari/amer-prototype/application/repository/accessport"
	"github.com/rfanazhari/amer-prototype/application/usecase/accessusecase"
	"github.com/rfanazhari/amer-prototype/application/usecase/executionusecase"
	"github.com/rfanazhari/amer-prototype/domain/aggregate/execution"
	"github.com/rfanazhari/amer-prototype/infrastructure/repository/accessrepository"
	"github.com/rfanazhari/amer-prototype/infrastructure/service/accesscredentials"
	"github.com/rfanazhari/amer-prototype/test/mocks"
	"github.com/rfanazhari/amer-prototype/test/testdata"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// revokeAccessCase names one real PostgreSQL assignment access revocation proof.
type revokeAccessCase struct {
	name   string
	verify func(t *testing.T, f executionFixture, hr accessport.AccountRecord, revoke accessusecase.RevokeAssignmentAccess)
}

// TestRevokeAssignmentAccessPostgres proves the marker commit, the lifecycle cancellation and their repeat behavior.
func TestRevokeAssignmentAccessPostgres(t *testing.T) {
	tests := []revokeAccessCase{
		{name: "nonterminal execution becomes cancelled and a repeat keeps it", verify: proveRevocationCancelsExecution},
		{name: "revocation without an execution succeeds", verify: proveRevocationWithoutExecution},
		{name: "reissue leaves a nonterminal execution running", verify: proveReissueDoesNotCancel},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newExecutionFixture(t)
			hr := insertACPAccount(t, f.db)
			session := testdata.AdministrativeSessionRecord(t)
			session.ID = uuid.NewString()
			session.AccountID = hr.ID
			session.ExpiresAt = time.Now().UTC().Add(time.Hour)
			current := mocks.NewCurrentAdministrativeSession(t)
			current.On("Resolve", mock.Anything).Return(session, hr, nil)
			revoke := accessusecase.RevokeAssignmentAccess{
				Assignments: accessrepository.Store{DB: f.db},
				Current:     current,
				Lifecycle:   executionusecase.CancelNonterminal{Assignments: f.store, Executions: f.store, Clock: wallClock{}},
				Clock:       wallClock{},
			}
			tt.verify(t, f, hr, revoke)
		})
	}
}

// executionState reads the stored state and terminal reason of the fixture assignment's execution.
func executionState(t *testing.T, f executionFixture) (string, string) {
	t.Helper()
	var row struct {
		State          string
		TerminalReason string
	}
	require.NoError(t, f.db.Raw(`SELECT e.state, COALESCE(e.terminal_reason, '') AS terminal_reason FROM agent_executions e
		JOIN interview_executions i ON i.id = e.id WHERE i.assignment_id = ?`, f.assignmentID).Scan(&row).Error)
	return row.State, row.TerminalReason
}

// proveRevocationCancelsExecution shows CANCELLED after revocation and an unchanged original instant after a repeat.
func proveRevocationCancelsExecution(t *testing.T, f executionFixture, _ accessport.AccountRecord, revoke accessusecase.RevokeAssignmentAccess) {
	ctx := context.Background()
	snap, fact := f.admit(t)
	require.NoError(t, f.store.Insert(ctx, snap, fact))
	first, err := revoke.Execute(ctx, f.assignmentID)
	require.NoError(t, err)
	state, reason := executionState(t, f)
	require.Equal(t, string(execution.StateCancelled), state)
	require.Equal(t, "access_revoked", reason)
	repeat, err := revoke.Execute(ctx, f.assignmentID)
	require.NoError(t, err)
	require.True(t, first.AccessRevokedAt.Equal(repeat.AccessRevokedAt))
	state, _ = executionState(t, f)
	require.Equal(t, string(execution.StateCancelled), state)
	require.EqualValues(t, 0, f.count(t, `SELECT COUNT(*) FROM interview_assignments WHERE id = ? AND completed_at IS NOT NULL`, f.assignmentID))
}

// proveRevocationWithoutExecution shows a marker write with nothing to cancel.
func proveRevocationWithoutExecution(t *testing.T, f executionFixture, _ accessport.AccountRecord, revoke accessusecase.RevokeAssignmentAccess) {
	got, err := revoke.Execute(context.Background(), f.assignmentID)
	require.NoError(t, err)
	require.False(t, got.AccessRevokedAt.IsZero())
	require.EqualValues(t, 1, f.count(t, `SELECT COUNT(*) FROM interview_assignments WHERE id = ? AND access_revoked_at IS NOT NULL`, f.assignmentID))
}

// proveReissueDoesNotCancel shows invitation issue leaves a nonterminal execution untouched.
func proveReissueDoesNotCancel(t *testing.T, f executionFixture, hr accessport.AccountRecord, revoke accessusecase.RevokeAssignmentAccess) {
	ctx := context.Background()
	snap, fact := f.admit(t)
	require.NoError(t, f.store.Insert(ctx, snap, fact))
	_, err := (accessusecase.IssueInvitation{
		Assignments: revoke.Assignments, Current: revoke.Current, Tokens: accesscredentials.Tokens{}, Clock: wallClock{},
	}).Execute(ctx, f.assignmentID)
	require.NoError(t, err)
	state, _ := executionState(t, f)
	require.Equal(t, string(execution.StateCreated), state)
	require.EqualValues(t, 0, f.count(t, `SELECT COUNT(*) FROM interview_assignments WHERE id = ? AND access_revoked_at IS NOT NULL`, f.assignmentID))
	t.Cleanup(func() {
		require.NoError(t, f.db.Exec(`DELETE FROM interview_invitations WHERE assignment_id = ?`, f.assignmentID).Error)
	})
	_ = hr
}
