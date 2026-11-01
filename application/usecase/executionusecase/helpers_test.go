package executionusecase_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/rfanazhari/amer-prototype/application/repository/executionport"
	"github.com/rfanazhari/amer-prototype/application/service"
	"github.com/rfanazhari/amer-prototype/domain/aggregate/execution"
	"github.com/rfanazhari/amer-prototype/pkg/errs"
	"github.com/rfanazhari/amer-prototype/test/mocks"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

const (
	assignmentID = "11111111-1111-4111-8111-111111111111"
	instanceID   = "22222222-2222-4222-8222-222222222222"
	revisionID   = "33333333-3333-4333-8333-333333333333"
	otherRevID   = "44444444-4444-4444-8444-444444444444"
	newID        = "55555555-5555-4555-8555-555555555555"
	existingID   = "66666666-6666-4666-8666-666666666666"
	requestKey   = "start-001"
	actorRef     = "candidate-interview"
)

// fixedNow is the trusted instant returned by the test clock, with sub-millisecond noise to prove truncation.
var fixedNow = time.Date(2026, 3, 4, 5, 6, 7, 123456789, time.FixedZone("WIB", 7*3600))

// trustedNow is fixedNow as the application must hand it to the domain: UTC at millisecond precision.
func trustedNow() time.Time { return fixedNow.UTC().Truncate(time.Millisecond) }

// testClock returns a mock clock that always reports fixedNow.
func testClock(t *testing.T) *mocks.Clock {
	t.Helper()
	clock := mocks.NewClock(t)
	clock.On("Now").Return(fixedNow).Maybe()
	return clock
}

// fixedID returns an identity minter that always yields id.
func fixedID(id string) func() string { return func() string { return id } }

// openRow is an assignment that is neither revoked nor completed.
func openRow() executionport.AssignmentRow {
	return executionport.AssignmentRow{AssignmentID: assignmentID, InstanceID: instanceID, RevisionID: revisionID}
}

// snapshotIn builds a stored snapshot for the shared test binding in the requested state.
func snapshotIn(t *testing.T, state execution.State) execution.Snapshot {
	t.Helper()
	created := trustedNow().Add(-time.Hour)
	e, snap, _, err := execution.NewExecution(existingID, execution.Binding{InstanceID: instanceID, RevisionID: revisionID}, created)
	require.NoError(t, err)
	if state == execution.StateCreated {
		return snap
	}
	_, _, err = e.Activate(1, actorRef, "op", 1, nil, created.Add(time.Minute))
	require.NoError(t, err)
	if state == execution.StateRunning {
		return e.Snapshot()
	}
	snap2, _, err := e.Cancel("access_revoked", actorRef, created.Add(2*time.Minute))
	require.NoError(t, err)
	return snap2
}

// lockRunsWith makes the assignment lock invoke the callback with row, as the real adapter does.
func lockRunsWith(lock *mocks.AssignmentLock, row executionport.AssignmentRow, wrap func(error) error) {
	lock.On("WithAssignment", mock.Anything, assignmentID, mock.Anything).Return(
		func(ctx context.Context, _ string, fn func(context.Context, executionport.AssignmentRow) error) error {
			err := fn(ctx, row)
			if wrap != nil {
				return wrap(err)
			}
			return err
		})
}

// requireCode asserts err is an execution error carrying the public code.
func requireCode(t *testing.T, err error, code string) {
	t.Helper()
	var coded *service.ExecutionError
	require.True(t, errors.As(err, &coded), "want execution error %s, got %v", code, err)
	require.Equal(t, code, coded.Code)
}

// requireTechnical asserts err is the technical class and not a business code.
func requireTechnical(t *testing.T, err error) {
	t.Helper()
	var technical errs.Technical
	require.True(t, errors.As(err, &technical), "want technical error, got %v", err)
	var coded *service.ExecutionError
	require.False(t, errors.As(err, &coded))
}
