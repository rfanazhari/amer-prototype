package interviewusecase_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/rfanazhari/amer-prototype/application/service"
	"github.com/rfanazhari/amer-prototype/application/usecase/interviewusecase"
	"github.com/rfanazhari/amer-prototype/domain/enum"
	"github.com/rfanazhari/amer-prototype/domain/interview"
	"github.com/rfanazhari/amer-prototype/test/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// TestWorkflowFailures verifies denial and partial-provider failure preserve earlier committed evidence.
func TestWorkflowFailures(t *testing.T) {
	// failureCase selects an orchestration failure with independently expected side effects.
	type failureCase struct {
		name     string
		scenario string
		wantErr  bool
	}
	tests := []failureCase{
		{name: "deadline reached during summary writes nothing", scenario: "deadline", wantErr: true},
		{name: "locked fence rejection invokes no write", scenario: "locked-fence", wantErr: true},
		{name: "missing handoff opens no session", scenario: "no-handoff", wantErr: true},
		{name: "provider failure keeps handoff", scenario: "create", wantErr: true},
		{name: "attach failure cleans provider and keeps handoff", scenario: "attach", wantErr: true},
		{name: "stale progress version opens no session", scenario: "version", wantErr: true},
		{name: "invalid authority opens no session", scenario: "invalid", wantErr: true},
		{name: "mismatched observed execution writes nothing", scenario: "observation", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := newHarness(t, true)
			var err error
			if tt.scenario == "deadline" {
				clock := mocks.NewClock(t)
				clock.On("Now").Return(fixedTime()).Once()
				clock.On("Now").Return(fixedTime().Add(15 * time.Minute)).Once()
				h.workflow.Clock = clock
				before := cloneRecord(t, h.record)
				err = h.workflow.Propose(context.Background(), interviewusecase.Proposal{Authority: authority(), ProgressVersion: 3, Target: enum.StageScreening})
				require.ErrorIs(t, err, interview.ErrConflict)
				assert.Equal(t, before, h.record)
				return
			}
			if tt.scenario == "locked-fence" {
				h.writeErr = interview.ErrConflict
				before := cloneRecord(t, h.record)
				err = h.workflow.Propose(context.Background(), interviewusecase.Proposal{Authority: authority(), ProgressVersion: 3, Target: enum.StageScreening})
				require.ErrorIs(t, err, interview.ErrConflict)
				assert.Equal(t, before, h.record)
				h.store.AssertNotCalled(t, "Update", mock.Anything, mock.Anything, mock.Anything)
				return
			}
			if tt.scenario == "observation" {
				before := cloneRecord(t, h.record)
				err = h.workflow.Disconnect(context.Background(), authority(), service.InterviewVoiceSession{ExecutionID: "another-execution", AttemptID: "old-attempt", SessionID: "old-session"})
				require.Error(t, err)
				assert.Equal(t, before, h.record)
				h.store.AssertNotCalled(t, "Update", mock.Anything, mock.Anything, mock.Anything)
				return
			}
			if tt.scenario != "no-handoff" {
				require.NoError(t, h.workflow.Propose(context.Background(), interviewusecase.Proposal{Authority: authority(), ProgressVersion: 3, Target: enum.StageScreening}))
			}
			r := request(t, h)
			switch tt.scenario {
			case "create":
				h.createErr = errors.New("provider failed")
			case "attach":
				h.attachErr = errors.New("attach failed")
			case "version":
				r.ProgressVersion--
			case "invalid":
				r.Authority.AttemptID = ""
			}
			_, err = h.workflow.OpenStage(context.Background(), r)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			switch tt.scenario {
			case "create", "attach":
				require.Len(t, h.record.Snapshot.Handoffs, 1)
				assert.Equal(t, []string{"answer"}, h.record.Snapshot.Handoffs[0].EntryIDs)
				assert.Nil(t, h.record.Sessions[1].ProviderSessionID)
				if tt.scenario == "attach" {
					h.driver.AssertNumberOfCalls(t, "Close", 1)
				}
			default:
				h.driver.AssertNotCalled(t, "Create", mock.Anything, mock.Anything)
			}
		})
	}
}
