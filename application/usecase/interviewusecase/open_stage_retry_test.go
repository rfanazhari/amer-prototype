package interviewusecase_test

import (
	"context"
	"errors"
	"testing"

	"github.com/rfanazhari/amer-prototype/application/usecase/interviewusecase"
	"github.com/rfanazhari/amer-prototype/domain/enum"
	"github.com/rfanazhari/amer-prototype/domain/interview"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// TestOpenStageRetry verifies a failed target create is retried against the committed target only.
func TestOpenStageRetry(t *testing.T) {
	// retryCase selects how the first target create fails and what the retry must observe.
	type retryCase struct {
		name       string
		firstFails string
	}
	tests := []retryCase{
		{name: "failed provider create retries the same target", firstFails: "create"},
		{name: "failed attach retries the same target", firstFails: "attach"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := newHarness(t, true)
			require.NoError(t, h.workflow.Propose(context.Background(), interviewusecase.Proposal{Authority: authority(), ProgressVersion: 3, Target: enum.StageScreening}))
			if tt.firstFails == "create" {
				h.createErr = errors.New("provider failed")
			} else {
				h.attachErr = errors.New("attach failed")
			}
			_, err := h.workflow.OpenStage(context.Background(), request(t, h))
			require.Error(t, err)
			h.createErr, h.attachErr = nil, nil

			retry := request(t, h)
			retry.SessionID = "retry-session"
			created, err := h.workflow.OpenStage(context.Background(), retry)

			require.NoError(t, err)
			assert.Equal(t, "execution", created.Session.ExecutionID)
			assert.Equal(t, enum.StageScreening, h.record.Snapshot.CurrentStage)
			require.Len(t, h.record.Snapshot.Handoffs, 1)
			h.lifecycle.AssertNotCalled(t, "Activate", mock.Anything, mock.Anything)
			var open []string
			for _, s := range h.record.Sessions {
				if s.ClosedAt == nil {
					open = append(open, s.ID)
					assert.Equal(t, enum.StageScreening, s.Stage)
					require.NotNil(t, s.ProviderSessionID)
				}
			}
			assert.Equal(t, []string{"retry-session"}, open)
		})
	}
}

// TestOpenStageRetryRefusals verifies a retry never replaces a provisioned session or a missing target.
func TestOpenStageRetryRefusals(t *testing.T) {
	h := newHarness(t, true)
	require.NoError(t, h.workflow.Propose(context.Background(), interviewusecase.Proposal{Authority: authority(), ProgressVersion: 3, Target: enum.StageScreening}))
	_, err := h.workflow.OpenStage(context.Background(), request(t, h))
	require.NoError(t, err)
	creates := len(h.creates)

	again := request(t, h)
	again.SessionID = "another-session"
	_, err = h.workflow.OpenStage(context.Background(), again)

	require.ErrorIs(t, err, interview.ErrConflict)
	assert.Len(t, h.creates, creates)
}
