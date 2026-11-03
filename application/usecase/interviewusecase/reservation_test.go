package interviewusecase_test

import (
	"context"
	"testing"

	"github.com/rfanazhari/amer-prototype/application/repository/interviewport"
	"github.com/rfanazhari/amer-prototype/application/usecase/interviewusecase"
	"github.com/rfanazhari/amer-prototype/domain/interview"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// TestAttemptReservationReplay refuses ambiguous setup and retained commands without inventing SDP.
func TestAttemptReservationReplay(t *testing.T) {
	// testCase pairs a stored state with its safe public failure.
	type testCase struct {
		name    string
		state   string
		wantErr error
	}
	tests := []testCase{
		{name: "pending has no successful answer", state: "pending", wantErr: interviewusecase.ErrAttemptNotReady},
		{name: "failed key cannot create again", state: "failed", wantErr: interviewusecase.ErrAttemptConflict},
		{name: "legacy key cannot invent an answer", state: "legacy", wantErr: interviewusecase.ErrAttemptConflict},
		{name: "corrupt success without original result is technical", state: "succeeded", wantErr: interview.ErrRestore},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := newAttemptHarness(t)
			h.session(t)
			h.owned(5, attemptExecution)
			h.controlled(t, attemptTab)
			h.commands.On("Find", mock.Anything, attemptExecution, storedKey()).Return(interviewport.SessionCommand{State: tt.state, Intent: "activate"}, nil)
			h.progress.On("Find", mock.Anything, attemptExecution).Return(interviewport.ProgressRecord{}, interviewport.ErrNotFound).Maybe()
			got, err := h.op.Execute(context.Background(), attemptInput())
			require.ErrorIs(t, err, tt.wantErr)
			require.Nil(t, got)
		})
	}
}
