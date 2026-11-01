package executionquery_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/rfanazhari/amer-prototype/application/query/executionquery"
	"github.com/rfanazhari/amer-prototype/application/repository/executionport"
	"github.com/rfanazhari/amer-prototype/application/service"
	"github.com/rfanazhari/amer-prototype/domain/aggregate/execution"
	"github.com/rfanazhari/amer-prototype/pkg/errs"
	"github.com/rfanazhari/amer-prototype/test/mocks"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

const (
	executionID  = "66666666-6666-4666-8666-666666666666"
	assignmentID = "11111111-1111-4111-8111-111111111111"
)

// runningSnapshot returns a stored RUNNING execution with an activation time.
func runningSnapshot(t *testing.T) execution.Snapshot {
	t.Helper()
	created := time.Date(2026, 3, 4, 5, 0, 0, 0, time.UTC)
	e, _, _, err := execution.NewExecution(executionID, execution.Binding{InstanceID: "inst", RevisionID: "rev"}, created)
	require.NoError(t, err)
	snap, _, err := e.Activate(1, "runtime", "op", 1, nil, created.Add(time.Minute))
	require.NoError(t, err)
	return snap
}

// TestFindReturnsDetachedViewWithoutWriting reads by id using only the store's Find.
func TestFindReturnsDetachedViewWithoutWriting(t *testing.T) {
	store := mocks.NewExecutionStore(t)
	snap := runningSnapshot(t)
	store.On("Find", mock.Anything, executionID).Return(snap, nil)

	view, err := executionquery.Find{Executions: store}.Execute(context.Background(), executionID)

	require.NoError(t, err)
	require.NotNil(t, view)
	require.Equal(t, executionID, view.ID)
	require.Equal(t, "RUNNING", view.State)
	require.Empty(t, view.AssignmentID)
	require.NotNil(t, view.ActivatedAt)
	require.Nil(t, view.TerminalAt)
	*view.ActivatedAt = view.ActivatedAt.Add(time.Hour)
	require.Equal(t, time.Date(2026, 3, 4, 5, 1, 0, 0, time.UTC), *snap.ActivatedAt, "view times must not alias the snapshot")
}

// TestFindMapsAbsenceAndTechnicalFailure covers EXL_NOT_FOUND and the technical class.
func TestFindMapsAbsenceAndTechnicalFailure(t *testing.T) {
	store := mocks.NewExecutionStore(t)
	store.On("Find", mock.Anything, executionID).Return(execution.Snapshot{}, executionport.ErrNotFound).Once()
	store.On("Find", mock.Anything, executionID).Return(execution.Snapshot{}, errors.New("boom")).Once()
	q := executionquery.Find{Executions: store}

	view, err := q.Execute(context.Background(), executionID)
	require.Nil(t, view)
	var coded *service.ExecutionError
	require.ErrorAs(t, err, &coded)
	require.Equal(t, errs.EXLNotFound, coded.Code)

	view, err = q.Execute(context.Background(), executionID)
	require.Nil(t, view)
	var technical errs.Technical
	require.ErrorAs(t, err, &technical)
}

// TestFindByAssignmentReturnsLinkedExecutionWithoutInserting reads through the pin and never creates one.
func TestFindByAssignmentReturnsLinkedExecutionWithoutInserting(t *testing.T) {
	store := mocks.NewExecutionStore(t)
	store.On("FindByAssignment", mock.Anything, assignmentID).Return(runningSnapshot(t), nil).Once()
	store.On("FindByAssignment", mock.Anything, assignmentID).Return(execution.Snapshot{}, executionport.ErrNotFound).Once()
	q := executionquery.FindByAssignment{Executions: store}

	view, err := q.Execute(context.Background(), assignmentID)
	require.NoError(t, err)
	require.NotNil(t, view)
	require.Equal(t, assignmentID, view.AssignmentID)
	require.Equal(t, executionID, view.ID)

	view, err = q.Execute(context.Background(), assignmentID)
	require.Nil(t, view)
	var coded *service.ExecutionError
	require.ErrorAs(t, err, &coded)
	require.Equal(t, errs.EXLNotFound, coded.Code)
}
