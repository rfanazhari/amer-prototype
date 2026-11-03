package interview_test

import (
	"testing"
	"time"

	"github.com/rfanazhari/amer-prototype/domain/enum"
	"github.com/rfanazhari/amer-prototype/domain/interview"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestProgressAttestSealsWithoutCoverage proves a model statement seals the stage and moves on without resolving coverage.
func TestProgressAttestSealsWithoutCoverage(t *testing.T) {
	p := progressFixture(t)
	at := instant().Add(time.Second)

	require.ErrorIs(t, p.Authorize(enum.StageScreening, at), interview.ErrConflict, "the evidence path still needs coverage")
	require.NoError(t, p.Attest(at))

	s := p.Snapshot()
	assert.Equal(t, enum.StageScreening, s.CurrentStage)
	assert.Equal(t, at, s.StageActivatedAt)
	require.Len(t, s.Handoffs, 1)
	assert.Equal(t, enum.StageBackground, s.Handoffs[0].Stage)
	assert.Equal(t, enum.StageScreening, s.Handoffs[0].Target)
	assert.Equal(t, at, s.Handoffs[0].CommittedAt)
	require.Len(t, s.Handoffs[0].Coverage, 1)
	assert.Equal(t, interview.Pending, s.Handoffs[0].Coverage[0].State, "attestation does not invent coverage")
	for _, item := range s.Coverage {
		assert.Equal(t, interview.Pending, item.State)
	}
	_, err := interview.RestoreProgress(s)
	require.NoError(t, err, "a stored attested handoff restores")
}

// TestProgressAttestWholeInterview proves four statements seal every stage and the last one has no successor.
func TestProgressAttestWholeInterview(t *testing.T) {
	p := progressFixture(t)
	assert.False(t, p.AllStagesSealed())
	for i, want := range []enum.QuestionStage{enum.StageScreening, enum.StageCompensation, enum.StageClosing, enum.StageClosing} {
		require.NoError(t, p.Attest(instant().Add(time.Duration(i+1)*time.Minute)))
		assert.Equal(t, want, p.Snapshot().CurrentStage)
		assert.Equal(t, i == 3, p.AllStagesSealed())
	}
	s := p.Snapshot()
	require.Len(t, s.Handoffs, 4)
	assert.Equal(t, enum.QuestionStage(""), s.Handoffs[3].Target)
	assert.False(t, p.ReadyForCompletion(), "strict readiness still needs resolved coverage")
	_, err := interview.RestoreProgress(s)
	require.NoError(t, err)
	require.ErrorIs(t, p.Attest(instant().Add(5*time.Minute)), interview.ErrConflict, "a sealed closing stage cannot be sealed again")
}

// TestProgressAttestTiming proves the statement honors stage activation and the inclusive fifteen-minute deadline.
func TestProgressAttestTiming(t *testing.T) {
	// attestTimingCase names the offset from the running instant at which the model attests.
	type attestTimingCase struct {
		name    string
		offset  time.Duration
		wantErr bool
	}
	tests := []attestTimingCase{
		{name: "at the activation instant", offset: 0},
		{name: "before activation", offset: -time.Millisecond, wantErr: true},
		{name: "one millisecond before the deadline", offset: 15*time.Minute - time.Millisecond},
		{name: "at the exact deadline", offset: 15 * time.Minute, wantErr: true},
		{name: "after the deadline", offset: 15*time.Minute + time.Millisecond, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := progressFixture(t)

			err := p.Attest(instant().Add(tt.offset))

			if tt.wantErr {
				require.ErrorIs(t, err, interview.ErrConflict)
				assert.Equal(t, enum.StageBackground, p.Snapshot().CurrentStage)
				assert.Empty(t, p.Snapshot().Handoffs)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, enum.StageScreening, p.Snapshot().CurrentStage)
		})
	}
}
