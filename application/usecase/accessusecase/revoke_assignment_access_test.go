package accessusecase_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/rfanazhari/amer-prototype/application/repository/accessport"
	"github.com/rfanazhari/amer-prototype/application/service"
	"github.com/rfanazhari/amer-prototype/application/usecase/accessusecase"
	"github.com/rfanazhari/amer-prototype/application/usecase/executionusecase"
	"github.com/rfanazhari/amer-prototype/domain/enum"
	"github.com/rfanazhari/amer-prototype/test/mocks"
	"github.com/rfanazhari/amer-prototype/test/testdata"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// revokeAssignmentCase names one HR assignment access revocation outcome.
type revokeAssignmentCase struct {
	name          string
	assignmentID  string
	resolveErr    error
	role          enum.Role
	facts         accessport.AssignmentFacts
	factsErr      error
	wantLock      bool
	wantMark      bool
	wantLifecycle bool
	lifecycleErr  error
	wantAt        time.Time
	wantErr       error
}

// TestRevokeAssignmentAccess covers authority order, the marker write, lifecycle cancellation after commit and failures.
func TestRevokeAssignmentAccess(t *testing.T) {
	earlier := testdata.Now().Add(-time.Hour)
	live := testdata.AssignmentFacts(t)
	alreadyRevoked := testdata.AssignmentFacts(t)
	alreadyRevoked.AccessRevoked = true
	alreadyRevoked.AccessRevokedAt = &earlier
	tests := []revokeAssignmentCase{
		{name: "first revoke stores the trusted instant then cancels", assignmentID: testdata.AssignmentID, role: enum.HR, facts: live, wantLock: true, wantMark: true, wantLifecycle: true, wantAt: testdata.Now()},
		{name: "repeat keeps the original instant and still cancels", assignmentID: testdata.AssignmentID, role: enum.HR, facts: alreadyRevoked, wantLock: true, wantLifecycle: true, wantAt: earlier},
		{name: "lifecycle failure after the marker is incomplete", assignmentID: testdata.AssignmentID, role: enum.HR, facts: live, wantLock: true, wantMark: true, wantLifecycle: true, lifecycleErr: errors.New("lifecycle down"), wantErr: service.ErrCancellationIncomplete},
		{name: "admin is forbidden before the assignment is read", assignmentID: testdata.AssignmentID, role: enum.Admin, wantErr: service.ErrForbidden},
		{name: "unbound session is unauthenticated before the lock", assignmentID: testdata.AssignmentID, resolveErr: service.ErrSessionUnbound, wantErr: service.ErrSessionUnbound},
		{name: "malformed id is rejected before the lock", assignmentID: "not-a-uuid", role: enum.HR, wantErr: service.ErrInvalidInput},
		{name: "missing assignment does not call lifecycle", assignmentID: testdata.AssignmentID, role: enum.HR, factsErr: accessport.ErrNotFound, wantLock: true, wantErr: accessport.ErrNotFound},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hr := testdata.AdministrativeAccountRecord(t)
			hr.Role = tt.role
			session := testdata.AdministrativeSessionRecord(t)
			session.AccountID = hr.ID
			assignments := mocks.NewAssignmentAccess(t)
			current := mocks.NewCurrentAdministrativeSession(t)
			lifecycle := mocks.NewExecutionCancelNonterminalOperation(t)
			clock := mocks.NewClock(t)
			clock.On("Now").Return(testdata.Now()).Maybe()
			current.On("Resolve", mock.Anything).Return(session, hr, tt.resolveErr).Once()
			committed := false
			if tt.wantLock {
				assignments.On("WithAssignment", mock.Anything, tt.assignmentID, mock.Anything).Return(func(ctx context.Context, _ string, fn func(accessport.AssignmentTx) error) error {
					tx := mocks.NewAssignmentTx(t)
					tx.On("Facts", mock.Anything).Return(tt.facts, tt.factsErr).Once()
					if tt.wantMark {
						tx.On("MarkAccessRevoked", mock.Anything, mock.MatchedBy(func(at time.Time) bool { return at.Equal(testdata.Now()) })).Return(nil).Once()
					}
					err := fn(tx)
					committed = err == nil
					return err
				}).Once()
			}
			if tt.wantLifecycle {
				lifecycle.On("Execute", mock.Anything, executionusecase.CancelNonterminalInput{
					AssignmentID: tt.assignmentID, ActorRef: hr.ID, Reason: "access_revoked",
				}).Run(func(mock.Arguments) { require.True(t, committed, "lifecycle must run after the access commit") }).Return(tt.lifecycleErr).Once()
			}
			got, err := (accessusecase.RevokeAssignmentAccess{
				Assignments: assignments, Current: current, Lifecycle: lifecycle, Clock: clock,
			}).Execute(context.Background(), tt.assignmentID)
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				assert.Nil(t, got)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, got)
			assert.Equal(t, tt.assignmentID, got.ID)
			assert.True(t, tt.wantAt.Equal(got.AccessRevokedAt))
		})
	}
}
