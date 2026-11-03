package interviewusecase_test

import (
	"context"
	"github.com/rfanazhari/amer-prototype/application/service"
	"github.com/rfanazhari/amer-prototype/domain/interview"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"testing"
)

// TestWorkflowDisconnectEpisodeExhaustion fails the same execution when a third backend-detected episode is refused.
func TestWorkflowDisconnectEpisodeExhaustion(t *testing.T) {
	// testCase selects a persisted recovery episode count.
	type testCase struct {
		name     string
		episodes int
		wantErr  bool
	}
	tests := []testCase{{name: "second episode remains available", episodes: 1}, {name: "third episode fails current execution", episodes: 2, wantErr: true}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := newHarness(t, false)
			p, err := interview.RestoreProgress(h.record.Snapshot)
			require.NoError(t, err)
			for range tt.episodes {
				require.NoError(t, p.BeginReconnect(fixedTime()))
				require.NoError(t, p.AttemptReconnect(fixedTime()))
				require.NoError(t, p.FinishReconnect())
			}
			h.record.Snapshot = p.Snapshot()
			if tt.wantErr {
				h.lifecycle.On("Fail", mock.Anything, service.InterviewFailure{Authority: authority(), Reason: "reconnect_budget_exhausted"}).Return(nil).Once()
			}
			err = h.workflow.Disconnect(context.Background(), authority(), service.InterviewVoiceSession{ExecutionID: "execution", AttemptID: "old-attempt", SessionID: "old-session"})
			if tt.wantErr {
				require.ErrorIs(t, err, interview.ErrBudget)
			} else {
				require.NoError(t, err)
			}
			require.Equal(t, 2, h.record.Snapshot.Reconnect.Episodes)
		})
	}
}
