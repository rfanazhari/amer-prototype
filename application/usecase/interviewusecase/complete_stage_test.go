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

// stageProgress builds progress through the evidence path until the target stage is current.
// The closing stage is also sealed when sealClosing is set.
func stageProgress(t *testing.T, target enum.QuestionStage, sealClosing bool) interview.Snapshot {
	t.Helper()
	p, err := interview.NewProgress("execution", pinnedRevision(t), finishRunning())
	require.NoError(t, err)
	at := finishRunning().Add(time.Second)
	for _, step := range []struct{ stage, next enum.QuestionStage }{{enum.StageBackground, enum.StageScreening}, {enum.StageScreening, enum.StageCompensation}, {enum.StageCompensation, enum.StageClosing}} {
		if p.Snapshot().CurrentStage == target {
			break
		}
		id := string(step.stage) + "-answer"
		require.NoError(t, p.Ask(string(step.stage)))
		require.NoError(t, p.RecordEntry(interview.TranscriptEntry{ID: id, Stage: step.stage, Speaker: interview.Candidate, State: interview.Final, Body: "Recorded answer"}))
		require.NoError(t, p.Answer(string(step.stage), id))
		require.NoError(t, p.CommitHandoff(step.next, at))
		require.NoError(t, p.ActivateStage(step.next, at))
	}
	if sealClosing {
		require.NoError(t, p.CommitHandoff("", at))
	}
	return p.Snapshot()
}

// TestWorkflowCompleteStage proves the model's statement advances, repeats, completes or is refused.
func TestWorkflowCompleteStage(t *testing.T) {
	running := finishRunning()
	rejected := errors.New("lifecycle rejected")
	// completeCase selects the stored stage, the statement, the trusted instant and the failing port.
	type completeCase struct {
		name          string
		current       enum.QuestionStage
		sealClosing   bool
		stage         enum.QuestionStage
		now           time.Time
		fenceErr      error
		writeErr      error
		completeErr   error
		wantOutcome   StageOutcomeWant
		wantErr       error
		wantStage     enum.QuestionStage
		wantHandoffs  int
		wantComplete  bool
		wantSessionOK bool
	}
	at := running.Add(5 * time.Minute)
	tests := []completeCase{
		{name: "background advances to screening", current: enum.StageBackground, stage: enum.StageBackground, now: at, wantOutcome: advanced, wantStage: enum.StageScreening, wantHandoffs: 1},
		{name: "compensation advances to closing", current: enum.StageCompensation, stage: enum.StageCompensation, now: at, wantOutcome: advanced, wantStage: enum.StageClosing, wantHandoffs: 3},
		{name: "closing completes the interview and closes the session", current: enum.StageClosing, stage: enum.StageClosing, now: at, wantOutcome: completed, wantStage: enum.StageClosing, wantHandoffs: 4, wantComplete: true, wantSessionOK: true},
		{name: "a repeated earlier stage changes nothing", current: enum.StageScreening, stage: enum.StageBackground, now: at, wantOutcome: repeated, wantStage: enum.StageScreening, wantHandoffs: 1},
		{name: "a repeated closing statement retries completion only", current: enum.StageClosing, sealClosing: true, stage: enum.StageClosing, now: at, wantOutcome: completed, wantStage: enum.StageClosing, wantHandoffs: 4, wantComplete: true},
		{name: "skipping a stage conflicts", current: enum.StageBackground, stage: enum.StageCompensation, now: at, wantErr: interview.ErrConflict, wantStage: enum.StageBackground},
		{name: "an unknown stage is invalid", current: enum.StageBackground, stage: enum.QuestionStage("bogus"), now: at, wantErr: interview.ErrInvalid, wantStage: enum.StageBackground},
		{name: "the exact hard deadline conflicts", current: enum.StageBackground, stage: enum.StageBackground, now: running.Add(15 * time.Minute), wantErr: interview.ErrConflict, wantStage: enum.StageBackground},
		{name: "a stale attempt writes nothing", current: enum.StageBackground, stage: enum.StageBackground, now: at, fenceErr: interview.ErrConflict, wantErr: interview.ErrConflict, wantStage: enum.StageBackground},
		{name: "a failed write leaves the stage current", current: enum.StageBackground, stage: enum.StageBackground, now: at, writeErr: rejected, wantErr: rejected, wantStage: enum.StageBackground},
		{name: "lifecycle rejection keeps the sealed record", current: enum.StageClosing, stage: enum.StageClosing, now: at, completeErr: rejected, wantErr: rejected, wantStage: enum.StageClosing, wantHandoffs: 4, wantComplete: true, wantSessionOK: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := newHarness(t, false)
			h.record.Snapshot = stageProgress(t, tt.current, tt.sealClosing)
			h.record.Version = 9
			h.fenceErr, h.writeErr = tt.fenceErr, tt.writeErr
			clock := mocks.NewClock(t)
			clock.On("Now").Return(tt.now).Maybe()
			h.workflow.Clock = clock
			if tt.wantComplete {
				h.lifecycle.On("Complete", mock.Anything, service.InterviewCompletion{Authority: authority(), ProgressReference: "execution"}).Return(tt.completeErr).Once()
			}

			outcome, err := h.workflow.CompleteStage(context.Background(), interviewusecase.CompleteStageRequest{Authority: authority(), Stage: tt.stage})

			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.wantOutcome.outcome(), outcome)
			}
			p, restoreErr := interview.RestoreProgress(h.record.Snapshot)
			require.NoError(t, restoreErr)
			assert.Equal(t, tt.wantStage, p.Snapshot().CurrentStage)
			if tt.wantHandoffs > 0 || tt.wantErr == nil {
				assert.Len(t, p.Snapshot().Handoffs, tt.wantHandoffs)
			}
			require.NotEmpty(t, h.record.Sessions)
			assert.Equal(t, tt.wantSessionOK, h.record.Sessions[0].ClosedAt != nil)
		})
	}
}

// StageOutcomeWant names the expected outcome in the table without importing constants into every row.
type StageOutcomeWant int

const (
	advanced StageOutcomeWant = iota + 1
	repeated
	completed
)

// outcome maps the table name to the application outcome.
func (w StageOutcomeWant) outcome() interviewusecase.StageOutcome {
	switch w {
	case advanced:
		return interviewusecase.StageAdvanced
	case repeated:
		return interviewusecase.StageRepeated
	}
	return interviewusecase.InterviewCompleted
}
