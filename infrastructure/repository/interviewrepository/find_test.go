package interviewrepository_test

import (
	"context"
	"testing"
	"time"

	"github.com/rfanazhari/amer-prototype/application/repository/interviewport"
	"github.com/rfanazhari/amer-prototype/domain/enum"
	"github.com/rfanazhari/amer-prototype/domain/interview"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// findCase names one read of a stored progress boundary.
type findCase struct {
	name     string
	scenario scenario
	wantErr  error
	check    func(t *testing.T, rec interviewport.ProgressRecord)
}

// TestStoreFind proves restoration, derived handoff evidence, absence and corrupt state.
func TestStoreFind(t *testing.T) {
	corrupt := freshScenario()
	corrupt.coverage[0][4] = "weird"
	countOnly := freshScenario()
	countOnly.coverage[0][6] = int64(1)
	tests := []findCase{
		{
			name:     "clarification, follow-up and reconnect state restore from their columns",
			scenario: clarifiedScenario(),
			check: func(t *testing.T, rec interviewport.ProgressRecord) {
				assert.Equal(t, interview.Asked, rec.Snapshot.Coverage[0].State)
				assert.Equal(t, entryTwo, rec.Snapshot.Coverage[0].ClarificationEntryID)
				assert.Equal(t, []string{entryOne}, rec.Snapshot.FollowUps)
				assert.Equal(t, interview.ReconnectState{Episodes: 1, Attempts: 1}, rec.Snapshot.Reconnect)
			},
		},
		{
			name:     "fresh progress restores pending coverage and the open session",
			scenario: freshScenario(),
			check: func(t *testing.T, rec interviewport.ProgressRecord) {
				assert.Equal(t, int64(3), rec.Version)
				assert.Nil(t, rec.ControllerID)
				assert.Equal(t, enum.StageBackground, rec.Snapshot.CurrentStage)
				assert.Equal(t, 2, rec.Snapshot.FollowUpLimit)
				assert.Equal(t, revisionID, rec.Snapshot.RevisionID)
				require.Len(t, rec.Snapshot.Coverage, 3)
				assert.Equal(t, interview.Pending, rec.Snapshot.Coverage[2].State)
				require.Len(t, rec.Sessions, 1)
				assert.Equal(t, sessionOne, rec.Sessions[0].ID)
				assert.Nil(t, rec.Sessions[0].ClosedAt)
			},
		},
		{
			name:     "sealed stage restores its handoff evidence, summary, entry stage and checkpoint",
			scenario: sealedScenario(),
			check: func(t *testing.T, rec interviewport.ProgressRecord) {
				assert.Equal(t, enum.StageScreening, rec.Snapshot.CurrentStage)
				require.Len(t, rec.Snapshot.Entries, 1)
				assert.Equal(t, interview.TranscriptEntry{ID: entryOne, Stage: enum.StageBackground, Speaker: interview.Candidate, State: interview.Final, Body: "I built payment APIs"}, rec.Snapshot.Entries[0])
				require.Len(t, rec.Snapshot.Handoffs, 1)
				handoff := rec.Snapshot.Handoffs[0]
				assert.Equal(t, enum.StageBackground, handoff.Stage)
				assert.Equal(t, enum.StageScreening, handoff.Target)
				assert.Equal(t, []string{entryOne}, handoff.EntryIDs)
				require.Len(t, handoff.Coverage, 1)
				assert.Equal(t, entryOne, handoff.Coverage[0].AnswerEntryID)
				assert.Equal(t, map[enum.QuestionStage]string{enum.StageBackground: "Covered background"}, rec.HandoffSummaries)
				require.Len(t, rec.Sessions, 2)
				require.NotNil(t, rec.Sessions[0].ClosedAt)
				assert.WithinDuration(t, base.Add(90*time.Second), *rec.Sessions[0].ClosedAt, 0)
				assert.Equal(t, []interviewport.Checkpoint{{ID: "checkpoint-1", Stage: enum.StageBackground, SessionID: sessionOne, AttemptID: attemptOne, EntryIDs: []string{entryOne}, CommittedAt: base.Add(time.Minute)}}, rec.Checkpoints)
			},
		},
		{name: "absent progress is not found", scenario: scenario{}, wantErr: interviewport.ErrNotFound},
		{name: "corrupt coverage state is a technical restore error", scenario: corrupt, wantErr: interview.ErrRestore},
		{name: "clarification count without an entry id is a technical restore error", scenario: countOnly, wantErr: interview.ErrRestore},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newFixture(t)
			f.mock.ExpectBegin()
			expectProgressRead(f.mock, tt.scenario, false)
			if len(tt.scenario.progress) == 0 {
				f.mock.ExpectRollback()
			} else {
				f.pins.On("Find", mock.Anything, execID).Return(pinned(t), nil).Once()
				expectChildrenRead(f.mock, tt.scenario)
				if tt.wantErr != nil {
					f.mock.ExpectRollback()
				} else {
					f.mock.ExpectCommit()
				}
			}

			rec, err := f.store.Find(context.Background(), execID)

			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				assert.NotErrorIs(t, err, interviewport.ErrConflict)
				return
			}
			require.NoError(t, err)
			tt.check(t, rec)
		})
	}
}
