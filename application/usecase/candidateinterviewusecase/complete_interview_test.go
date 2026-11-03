package candidateinterviewusecase_test

import (
	"context"
	"errors"
	"testing"

	"github.com/rfanazhari/amer-prototype/application/service"
	"github.com/rfanazhari/amer-prototype/application/usecase/candidateinterviewusecase"
	"github.com/rfanazhari/amer-prototype/domain/aggregate/candidateinterview"
	"github.com/rfanazhari/amer-prototype/test/mocks"
	"github.com/rfanazhari/amer-prototype/test/testdata"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// completeGraph wires CompleteInterview to the shared strict mocks and a strict completer.
func completeGraph(t *testing.T) (*orchGraph, *mocks.InterviewCompleter, candidateinterviewusecase.CompleteInterview) {
	t.Helper()
	g := newOrchGraph(t)
	completer := mocks.NewInterviewCompleter(t)
	return g, completer, candidateinterviewusecase.CompleteInterview{Current: g.current, Controllers: g.controllers, Lifecycle: g.lifecycle, Completer: completer}
}

// TestCompleteInterviewByControllerCompletes proves the controlling tab ends a running interview as COMPLETED.
func TestCompleteInterviewByControllerCompletes(t *testing.T) {
	g, completer, op := completeGraph(t)
	completed := facts("COMPLETED")
	completed.Version = 4
	g.lifecycle.On("Find", mock.Anything, testdata.AssignmentID).Return(facts("RUNNING"), nil).Once()
	g.controllers.On("Find", mock.Anything, testdata.AssignmentID).Return(controllerSnap(t, orchTab, 2), nil).Once()
	completer.On("CompleteInterview", mock.Anything, orchExecutionID, int64(1)).Return(nil).Once()
	g.lifecycle.On("Find", mock.Anything, testdata.AssignmentID).Return(completed, nil).Once()

	got, err := op.Execute(context.Background(), candidateinterviewusecase.CompleteInterviewInput{ExecutionID: orchExecutionID, ControllerID: orchTab, ExpectedVersion: 1})

	require.NoError(t, err)
	assert.Equal(t, "COMPLETED", got.State)
	assert.EqualValues(t, 4, got.Version)
	assert.True(t, got.IsCurrentController)
	assert.EqualValues(t, 2, got.ControllerVersion)
}

// TestCompleteInterviewRepeatIsIdempotent proves a finished interview returns its view without completing again.
func TestCompleteInterviewRepeatIsIdempotent(t *testing.T) {
	g, _, op := completeGraph(t)
	completed := facts("COMPLETED")
	completed.Version = 4
	g.lifecycle.On("Find", mock.Anything, testdata.AssignmentID).Return(completed, nil).Once()
	g.controllers.On("Find", mock.Anything, testdata.AssignmentID).Return(controllerSnap(t, orchTab, 2), nil).Once()

	got, err := op.Execute(context.Background(), candidateinterviewusecase.CompleteInterviewInput{ExecutionID: orchExecutionID, ControllerID: orchTab, ExpectedVersion: 1})

	require.NoError(t, err)
	assert.Equal(t, "COMPLETED", got.State)
}

// TestCompleteInterviewRefusals proves each refusal happens before the completer is called.
func TestCompleteInterviewRefusals(t *testing.T) {
	failed := errors.New("completer failed")
	// refusalCase selects the stored execution and controller and the expected refusal.
	type refusalCase struct {
		name       string
		state      string
		execution  string
		controller string
		invalidTab bool
		findErr    error
		wantErr    error
	}
	tests := []refusalCase{
		{name: "another tab is not the controller", state: "RUNNING", controller: "tab-2", wantErr: candidateinterviewusecase.ErrNotController},
		{name: "a created interview has nothing to complete", state: "CREATED", wantErr: service.ErrInterviewNotLive},
		{name: "a cancelled interview is not relabelled", state: "CANCELLED", wantErr: candidateinterview.ErrExecutionTerminal},
		{name: "a failed interview is not relabelled", state: "FAILED", wantErr: candidateinterview.ErrExecutionTerminal},
		{name: "an execution outside the assignment is not found", state: "RUNNING", execution: "33333333-3333-4333-8333-333333333333", wantErr: service.ErrInterviewNotFound},
		{name: "an invalid tab is rejected before any port", invalidTab: true, wantErr: candidateinterview.ErrInvalidControllerID},
		{name: "an unbound lifecycle port", findErr: service.ErrInterviewDependencyUnavailable, wantErr: service.ErrInterviewDependencyUnavailable},
		{name: "a failing completer returns its error", state: "RUNNING", controller: orchTab, wantErr: failed},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g, completer, op := completeGraph(t)
			tab, controller, execution := orchTab, tt.controller, tt.execution
			if tt.invalidTab {
				tab = ""
			}
			if controller == "" {
				controller = orchTab
			}
			if execution == "" {
				execution = orchExecutionID
			}
			if tab != "" {
				if tt.findErr != nil {
					g.lifecycle.On("Find", mock.Anything, testdata.AssignmentID).Return(service.ExecutionFacts{}, tt.findErr).Once()
				} else {
					g.lifecycle.On("Find", mock.Anything, testdata.AssignmentID).Return(facts(tt.state), nil).Once()
				}
				if tt.state == "RUNNING" && execution == orchExecutionID {
					g.controllers.On("Find", mock.Anything, testdata.AssignmentID).Return(controllerSnap(t, controller, 2), nil).Once()
					if errors.Is(tt.wantErr, failed) {
						completer.On("CompleteInterview", mock.Anything, orchExecutionID, int64(1)).Return(failed).Once()
					}
				}
			}

			got, err := op.Execute(context.Background(), candidateinterviewusecase.CompleteInterviewInput{ExecutionID: execution, ControllerID: tab, ExpectedVersion: 1})

			assert.Nil(t, got)
			assert.ErrorIs(t, err, tt.wantErr)
		})
	}
}
