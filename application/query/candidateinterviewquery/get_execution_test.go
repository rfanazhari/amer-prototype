package candidateinterviewquery_test

import (
	"context"
	"reflect"
	"strings"
	"testing"

	"github.com/rfanazhari/amer-prototype/application/dto"
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

// executionID is the execution lifecycle reports in these scenarios.
const executionID = "22222222-2222-4222-8222-222222222222"

// getExecution builds GetExecution with strict mocks and a resolvable session.
func getExecution(t *testing.T) (candidateinterviewquery.GetExecution, *mocks.ExecutionLifecycle, *mocks.Controllers) {
	t.Helper()
	current := mocks.NewCurrentCandidateSession(t)
	current.On("Resolve", mock.Anything).Return(testdata.CandidateSessionRecord(t), nil).Maybe()
	lifecycle, controllers := mocks.NewExecutionLifecycle(t), mocks.NewControllers(t)
	return candidateinterviewquery.GetExecution{Current: current, Controllers: controllers, Lifecycle: lifecycle}, lifecycle, controllers
}

// controllerOf returns a stored controller for tab at version 2.
func controllerOf(t *testing.T, tab string) candidateinterview.ControllerSnapshot {
	t.Helper()
	id, err := candidateinterview.ParseControllerID(tab)
	require.NoError(t, err)
	return candidateinterview.ControllerSnapshot{AssignmentID: testdata.AssignmentID, ExecutionID: executionID, ControllerID: id, Version: 2, UpdatedAt: testdata.Now()}
}

// TestGetExecutionReadsWithoutCommitOrFind returns the controller-aware view through Find only.
func TestGetExecutionReadsWithoutCommitOrFind(t *testing.T) {
	op, lifecycle, controllers := getExecution(t)
	lifecycle.On("Find", mock.Anything, testdata.AssignmentID).Return(service.ExecutionFacts{ID: executionID, AssignmentID: testdata.AssignmentID, State: "RUNNING", Version: 3}, nil).Once()
	controllers.On("Find", mock.Anything, testdata.AssignmentID).Return(controllerOf(t, tabID), nil).Once()

	got, err := op.Execute(context.Background(), candidateinterviewquery.GetExecutionInput{ExecutionID: executionID, ControllerID: tabID})

	require.NoError(t, err)
	assert.Equal(t, executionID, got.ID)
	assert.Equal(t, "RUNNING", got.State)
	assert.EqualValues(t, 3, got.Version)
	assert.True(t, got.IsCurrentController)
	assert.EqualValues(t, 2, got.ControllerVersion)
	lifecycle.AssertNotCalled(t, "CommitOrFind", mock.Anything, mock.Anything)
}

// TestGetExecutionOtherTabIsNotController reports the second tab as read-only.
func TestGetExecutionOtherTabIsNotController(t *testing.T) {
	op, lifecycle, controllers := getExecution(t)
	lifecycle.On("Find", mock.Anything, testdata.AssignmentID).Return(service.ExecutionFacts{ID: executionID, State: "CREATED", Version: 1}, nil).Once()
	controllers.On("Find", mock.Anything, testdata.AssignmentID).Return(controllerOf(t, "tab-2"), nil).Once()

	got, err := op.Execute(context.Background(), candidateinterviewquery.GetExecutionInput{ExecutionID: executionID, ControllerID: tabID})

	require.NoError(t, err)
	assert.False(t, got.IsCurrentController)
}

// TestGetExecutionOutsideAssignmentIsNotFound hides an execution id that is not this assignment's.
func TestGetExecutionOutsideAssignmentIsNotFound(t *testing.T) {
	t.Run("different id", func(t *testing.T) {
		op, lifecycle, _ := getExecution(t)
		lifecycle.On("Find", mock.Anything, testdata.AssignmentID).Return(service.ExecutionFacts{ID: executionID, State: "RUNNING"}, nil).Once()
		got, err := op.Execute(context.Background(), candidateinterviewquery.GetExecutionInput{ExecutionID: "33333333-3333-4333-8333-333333333333"})
		assert.Nil(t, got)
		assert.ErrorIs(t, err, service.ErrInterviewNotFound)
	})
	t.Run("no execution", func(t *testing.T) {
		op, lifecycle, _ := getExecution(t)
		lifecycle.On("Find", mock.Anything, testdata.AssignmentID).Return(service.ExecutionFacts{}, service.ErrInterviewNotFound).Once()
		got, err := op.Execute(context.Background(), candidateinterviewquery.GetExecutionInput{ExecutionID: executionID})
		assert.Nil(t, got)
		assert.ErrorIs(t, err, service.ErrInterviewNotFound)
	})
	t.Run("unbound lifecycle", func(t *testing.T) {
		op, lifecycle, _ := getExecution(t)
		lifecycle.On("Find", mock.Anything, testdata.AssignmentID).Return(service.ExecutionFacts{}, service.ErrInterviewDependencyUnavailable).Once()
		got, err := op.Execute(context.Background(), candidateinterviewquery.GetExecutionInput{ExecutionID: executionID})
		assert.Nil(t, got)
		assert.ErrorIs(t, err, service.ErrInterviewDependencyUnavailable)
	})
}

// TestGetExecutionTerminalViewHasNoReportField renders CANCELLED without controller store failures and without report fields.
func TestGetExecutionTerminalViewHasNoReportField(t *testing.T) {
	op, lifecycle, controllers := getExecution(t)
	lifecycle.On("Find", mock.Anything, testdata.AssignmentID).Return(service.ExecutionFacts{ID: executionID, State: "CANCELLED", Version: 4}, nil).Once()
	controllers.On("Find", mock.Anything, testdata.AssignmentID).Return(candidateinterview.ControllerSnapshot{}, candidateinterviewport.ErrNotFound).Once()

	got, err := op.Execute(context.Background(), candidateinterviewquery.GetExecutionInput{ExecutionID: executionID, ControllerID: tabID})

	require.NoError(t, err)
	assert.Equal(t, "CANCELLED", got.State)
	assert.False(t, got.IsCurrentController)
	typ := reflect.TypeOf(dto.InterviewExecutionView{})
	for i := 0; i < typ.NumField(); i++ {
		name := strings.ToLower(typ.Field(i).Name)
		assert.NotContains(t, name, "report")
		assert.NotContains(t, name, "salary")
		assert.NotContains(t, name, "transcript")
	}
	_, hasControllerID := typ.FieldByName("ControllerID")
	assert.False(t, hasControllerID)
}
