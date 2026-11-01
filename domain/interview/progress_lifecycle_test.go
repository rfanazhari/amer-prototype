package interview_test

import (
	"testing"
	"time"

	"github.com/rfanazhari/amer-prototype/domain/enum"
	"github.com/rfanazhari/amer-prototype/domain/interview"
	"github.com/rfanazhari/amer-prototype/domain/valueobject"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestStageHandoffs proves authorization is pure and each target needs its sealed source.
func TestStageHandoffs(t *testing.T) {
	p := progressFixture(t)
	before := p.Snapshot()
	require.Error(t, p.Authorize(enum.StageScreening, instant()))
	require.Error(t, p.ActivateStage(enum.StageScreening, instant()))
	assert.Equal(t, before, p.Snapshot())
	answerFixture(t, p)
	before = p.Snapshot()
	require.Error(t, p.Authorize(enum.StageCompensation, instant()))
	require.NoError(t, p.Authorize(enum.StageScreening, instant()))
	assert.Equal(t, before, p.Snapshot())
	require.Error(t, p.ActivateStage(enum.StageScreening, instant()))
	// stageCase follows the fixed order and each revision-owned soft budget.
	type stageCase struct {
		name    string
		target  enum.QuestionStage
		seconds int
	}
	for _, tt := range []stageCase{{"screening", enum.StageScreening, 240}, {"compensation", enum.StageCompensation, 90}, {"closing", enum.StageClosing, 30}} {
		t.Run(tt.name, func(t *testing.T) {
			require.NoError(t, p.CommitHandoff(tt.target, instant().Add(time.Second)))
			assert.False(t, p.ReadyForCompletion())
			before := p.Snapshot()
			require.Error(t, p.CommitHandoff(tt.target, instant().Add(time.Second)))
			assert.Equal(t, before, p.Snapshot())
			require.NoError(t, p.ActivateStage(tt.target, instant().Add(time.Second)))
			assert.Equal(t, tt.target, p.Snapshot().CurrentStage)
			assert.False(t, p.SoftBudgetReached(instant().Add(time.Second+time.Duration(tt.seconds)*time.Second-time.Millisecond)))
			assert.True(t, p.SoftBudgetReached(instant().Add(time.Second+time.Duration(tt.seconds)*time.Second)))
			if tt.target != enum.StageClosing {
				answerFixture(t, p)
			}
		})
	}
	assert.False(t, p.ReadyForCompletion())
	require.NoError(t, p.Authorize("", instant().Add(time.Second)))
	require.NoError(t, p.CommitHandoff("", instant().Add(time.Second)))
	assert.True(t, p.ReadyForCompletion())
	s := p.Snapshot()
	require.Len(t, s.Handoffs, 4)
	assert.Empty(t, s.Handoffs[3].Target)
	before = p.Snapshot()
	require.Error(t, p.ActivateStage(enum.StageBackground, instant().Add(time.Second)))
	require.Error(t, p.RecordEntry(entryFixture("late", enum.StageClosing, interview.Candidate, interview.Final)))
	assert.Equal(t, before, p.Snapshot())
	restored, err := interview.RestoreProgress(before)
	require.NoError(t, err)
	assert.True(t, restored.ReadyForCompletion())
}

// TestDeadlineBoundaries proves equality uses the trusted running and activation instants.
func TestDeadlineBoundaries(t *testing.T) {
	// deadlineCase describes independent expected time predicates.
	type deadlineCase struct {
		name       string
		elapsed    time.Duration
		soft, hard bool
		wantErr    bool
	}
	for _, tt := range []deadlineCase{
		{"before soft", time.Minute - time.Millisecond, false, false, true},
		{"equal soft", time.Minute, true, false, false},
		{"before hard", 15*time.Minute - time.Millisecond, true, false, false},
		{"equal hard", 15 * time.Minute, true, true, true},
		{"after hard", 15*time.Minute + time.Millisecond, true, true, true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			p := progressFixture(t)
			at := instant().Add(tt.elapsed)
			assert.Equal(t, tt.soft, p.SoftBudgetReached(at))
			assert.Equal(t, tt.hard, p.HardDeadlineReached(at))
			before := p.Snapshot()
			err := p.Omit("background", interview.SoftBudgetExpired, "", at)
			if tt.wantErr {
				require.Error(t, err)
				assert.Equal(t, before, p.Snapshot())
			} else {
				require.NoError(t, err)
				require.NoError(t, p.Authorize(enum.StageScreening, at))
			}
			assert.False(t, p.ReadyForCompletion())
		})
	}
}

// TestReconnectBudgets proves the window, attempts, episodes and failure atomicity.
func TestReconnectBudgets(t *testing.T) {
	// reconnectCase supplies time since backend detection.
	type reconnectCase struct {
		name    string
		delay   time.Duration
		wantErr bool
	}
	for _, tt := range []reconnectCase{
		{"before detection", -time.Millisecond, true},
		{"at detection", 0, false},
		{"within window", 120*time.Second - time.Millisecond, false},
		{"equal window", 120 * time.Second, true},
		{"after window", 120*time.Second + time.Millisecond, true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			p := progressFixture(t)
			require.Error(t, p.AttemptReconnect(instant()))
			require.NoError(t, p.BeginReconnect(instant()))
			before := p.Snapshot()
			require.Error(t, p.BeginReconnect(instant()))
			assert.Equal(t, before, p.Snapshot())
			at := instant().Add(tt.delay)
			err := p.AttemptReconnect(at)
			if tt.wantErr {
				require.ErrorIs(t, err, interview.ErrBudget)
				assert.Equal(t, before, p.Snapshot())
				return
			}
			require.NoError(t, err)
			require.NoError(t, p.AttemptReconnect(at))
			before = p.Snapshot()
			require.ErrorIs(t, p.AttemptReconnect(at), interview.ErrBudget)
			assert.Equal(t, before, p.Snapshot())
			require.NoError(t, p.FinishReconnect())
			require.NoError(t, p.BeginReconnect(instant().Add(time.Second)))
			require.NoError(t, p.AttemptReconnect(instant().Add(time.Second)))
			assert.Equal(t, 2, p.Snapshot().Reconnect.Episodes)
			assert.Equal(t, 1, p.Snapshot().Reconnect.Attempts)
			require.NoError(t, p.FinishReconnect())
			before = p.Snapshot()
			require.ErrorIs(t, p.BeginReconnect(instant().Add(2*time.Second)), interview.ErrBudget)
			assert.Equal(t, before, p.Snapshot())
			assert.Equal(t, "execution", p.Snapshot().ExecutionID)
		})
	}
	p := progressFixture(t)
	require.ErrorIs(t, p.BeginReconnect(instant().Add(15*time.Minute)), interview.ErrBudget)
}

// TestRestoreProgress proves stored policy is not recomputed and corruption is technical.
func TestRestoreProgress(t *testing.T) {
	// restoreCase mutates a detached snapshot as a persistence reader would.
	type restoreCase struct {
		name    string
		change  func(*interview.Snapshot)
		wantErr bool
	}
	for _, tt := range []restoreCase{
		{"round trip", func(*interview.Snapshot) {}, false},
		{"historical cap", func(s *interview.Snapshot) { s.FollowUpLimit = 3 }, false},
		{"historical publication", func(s *interview.Snapshot) {
			c, err := valueobject.NewHRConfiguration(valueobject.HRInput{MandatoryQuestions: []valueobject.QuestionInput{{ID: "background", Stage: "background", Text: "Work?"}, {ID: "screening", Stage: "screening", Text: "Projects?"}, {ID: "compensation", Stage: "compensation", Text: "Expectations?"}}, StageSeconds: &valueobject.StageSecondsInput{Background: 10, Screening: 10, Compensation: 10, Closing: 10}, CompensationPolicy: &valueobject.CompensationInput{DiscussionMode: "expectations_only"}})
			require.NoError(t, err)
			s.Configuration = c
		}, false},
		{"missing execution", func(s *interview.Snapshot) { s.ExecutionID = "" }, true},
		{"missing revision", func(s *interview.Snapshot) { s.RevisionID = "" }, true},
		{"unknown stage", func(s *interview.Snapshot) { s.CurrentStage = "unknown" }, true},
		{"zero cap", func(s *interview.Snapshot) { s.FollowUpLimit = 0 }, true},
		{"unknown coverage", func(s *interview.Snapshot) { s.Coverage[0].State = "unknown" }, true},
		{"missing coverage", func(s *interview.Snapshot) { s.Coverage = s.Coverage[:1] }, true},
		{"wrong question stage", func(s *interview.Snapshot) { s.Coverage[0].Stage = enum.StageClosing }, true},
		{"answered without citation", func(s *interview.Snapshot) { s.Coverage[0].State = interview.Answered }, true},
		{"citation to fragment", func(s *interview.Snapshot) { s.Entries[0].State = interview.Fragment }, true},
		{"unknown omission", func(s *interview.Snapshot) {
			s.Coverage[1].State = interview.Omitted
			s.Coverage[1].OmissionReason = "technical_loss"
		}, true},
		{"foreign citation", func(s *interview.Snapshot) { s.Coverage[0].AnswerEntryID = "missing" }, true},
		{"counter corruption", func(s *interview.Snapshot) { s.Reconnect.Attempts = 3 }, true},
		{"unknown evidence", func(s *interview.Snapshot) { s.Entries[0].State = "unknown" }, true},
		{"unknown speaker", func(s *interview.Snapshot) { s.Entries[0].Speaker = "unknown" }, true},
		{"missing source handoff", func(s *interview.Snapshot) { s.CurrentStage = enum.StageScreening }, true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			p := progressFixture(t)
			answerFixture(t, p)
			s := p.Snapshot()
			tt.change(&s)
			restored, err := interview.RestoreProgress(s)
			if tt.wantErr {
				require.ErrorIs(t, err, interview.ErrRestore)
				assert.NotErrorIs(t, err, interview.ErrInvalid)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, s, restored.Snapshot())
		})
	}
}

// TestSnapshotOwnership proves restored and observed state cannot mutate aggregate invariants.
func TestSnapshotOwnership(t *testing.T) {
	p := progressFixture(t)
	require.NoError(t, p.RecordTechnicalLoss("loss", "background"))
	answerFixture(t, p)
	require.NoError(t, p.CommitHandoff(enum.StageScreening, instant()))
	require.NoError(t, p.ActivateStage(enum.StageScreening, instant()))
	require.NoError(t, p.BeginReconnect(instant()))
	s := p.Snapshot()
	restored, err := interview.RestoreProgress(s)
	require.NoError(t, err)
	before := restored.Snapshot()
	s.Coverage[0].State = interview.Pending
	s.Entries[0].Body = "mutated"
	s.Losses[0].Resolved = false
	s.Handoffs[0].Coverage[0].State = interview.Pending
	s.Handoffs[0].EntryIDs[0] = "missing"
	s.Handoffs[0].Losses[0].Resolved = false
	*s.Reconnect.DetectedAt = time.Time{}
	assert.Equal(t, before, restored.Snapshot())
	assert.Equal(t, before, p.Snapshot())
}
