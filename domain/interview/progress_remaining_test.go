package interview_test

import (
	"testing"

	"github.com/rfanazhari/amer-prototype/domain/enum"
	"github.com/rfanazhari/amer-prototype/domain/interview"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestProgressRemainingFollowUps proves the stored per-stage cap is reduced only by cited follow-ups of the current stage.
func TestProgressRemainingFollowUps(t *testing.T) {
	// remainingCase names how many follow-ups were consumed in each stage before the query.
	type remainingCase struct {
		name            string
		backgroundUsed  int
		screeningUsed   int
		advanceToScreen bool
		expected        int
	}
	tests := []remainingCase{
		{name: "fresh stage has the full cap", expected: 2},
		{name: "one follow-up consumed", backgroundUsed: 1, expected: 1},
		{name: "cap exhausted", backgroundUsed: 2, expected: 0},
		{name: "earlier stage usage does not reduce the next stage", backgroundUsed: 2, advanceToScreen: true, expected: 2},
		{name: "current stage usage after advance", backgroundUsed: 1, screeningUsed: 1, advanceToScreen: true, expected: 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := progressFixture(t)
			require.NoError(t, p.RecordEntry(entryFixture("background-evidence", enum.StageBackground, interview.Candidate, interview.Final)))
			for i := 0; i < tt.backgroundUsed; i++ {
				require.NoError(t, p.FollowUp("background-evidence"))
			}
			if tt.advanceToScreen {
				answerFixture(t, p)
				require.NoError(t, p.CommitHandoff(enum.StageScreening, instant()))
				require.NoError(t, p.ActivateStage(enum.StageScreening, instant()))
				require.NoError(t, p.RecordEntry(entryFixture("screening-evidence", enum.StageScreening, interview.Candidate, interview.Final)))
				for i := 0; i < tt.screeningUsed; i++ {
					require.NoError(t, p.FollowUp("screening-evidence"))
				}
			}
			assert.Equal(t, tt.expected, p.RemainingFollowUps())
		})
	}
}
