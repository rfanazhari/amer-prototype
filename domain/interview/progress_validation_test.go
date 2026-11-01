package interview_test

import (
	"testing"
	"time"

	"github.com/rfanazhari/amer-prototype/domain/enum"
	"github.com/rfanazhari/amer-prototype/domain/interview"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestRejectedOperations proves malformed observations and foreign questions leave progress unchanged.
func TestRejectedOperations(t *testing.T) {
	// rejectionCase exercises independent aggregate entry points.
	type rejectionCase struct {
		name    string
		run     func(*interview.Progress) error
		wantErr bool
	}
	for _, tt := range []rejectionCase{
		{"unknown question", func(p *interview.Progress) error { return p.Ask("unknown") }, true},
		{"future stage", func(p *interview.Progress) error { return p.Ask("screening") }, true},
		{"unknown loss question", func(p *interview.Progress) error { return p.RecordTechnicalLoss("loss", "unknown") }, true},
		{"empty loss id", func(p *interview.Progress) error { return p.RecordTechnicalLoss("", "background") }, true},
		{"clarification before ask", func(p *interview.Progress) error { return p.Clarify("background", "model") }, true},
		{"missing clarification", func(p *interview.Progress) error { return p.Clarify("background", "missing") }, true},
		{"fragment clarification", func(p *interview.Progress) error { return p.Clarify("background", "fragment") }, true},
		{"candidate clarification", func(p *interview.Progress) error { return p.Clarify("background", "candidate") }, true},
		{"entry missing id", func(p *interview.Progress) error {
			return p.RecordEntry(entryFixture("", enum.StageBackground, interview.Candidate, interview.Final))
		}, true},
		{"entry wrong stage", func(p *interview.Progress) error {
			return p.RecordEntry(entryFixture("wrong", enum.StageScreening, interview.Candidate, interview.Final))
		}, true},
		{"entry duplicate", func(p *interview.Progress) error {
			return p.RecordEntry(entryFixture("candidate", enum.StageBackground, interview.Candidate, interview.Final))
		}, true},
		{"entry unknown speaker", func(p *interview.Progress) error {
			return p.RecordEntry(entryFixture("unknown", enum.StageBackground, "system", interview.Final))
		}, true},
		{"entry unknown state", func(p *interview.Progress) error {
			return p.RecordEntry(entryFixture("unknown", enum.StageBackground, interview.Candidate, "unknown"))
		}, true},
		{"empty citation", func(p *interview.Progress) error { return p.FollowUp("") }, true},
		{"self proposal", func(p *interview.Progress) error { return p.Authorize(enum.StageBackground, instant()) }, true},
		{"early closing", func(p *interview.Progress) error { return p.Authorize("", instant()) }, true},
		{"hard deadline proposal", func(p *interview.Progress) error {
			return p.Authorize(enum.StageScreening, instant().Add(15*time.Minute))
		}, true},
		{"inactive reconnect finish", func(p *interview.Progress) error { return p.FinishReconnect() }, true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			p := progressFixture(t)
			require.NoError(t, p.RecordEntry(entryFixture("model", enum.StageBackground, interview.Model, interview.Final)))
			require.NoError(t, p.RecordEntry(entryFixture("candidate", enum.StageBackground, interview.Candidate, interview.Final)))
			require.NoError(t, p.RecordEntry(entryFixture("fragment", enum.StageBackground, interview.Model, interview.Fragment)))
			// Clarification cases need an asked question before the rejected call.
			if tt.name == "missing clarification" || tt.name == "fragment clarification" || tt.name == "candidate clarification" {
				require.NoError(t, p.Ask("background"))
			}
			before := p.Snapshot()
			err := tt.run(p)
			require.Error(t, err)
			assert.Equal(t, before, p.Snapshot())
		})
	}
}

// TestRestoreHandoffCorruption proves sealed snapshots and stage order cannot be forged.
func TestRestoreHandoffCorruption(t *testing.T) {
	// corruptionCase selects one invalid stored relationship.
	type corruptionCase struct {
		name    string
		change  func(*interview.Snapshot)
		wantErr bool
	}
	for _, tt := range []corruptionCase{
		{"wrong target", func(s *interview.Snapshot) { s.Handoffs[0].Target = enum.StageClosing }, true},
		{"missing record", func(s *interview.Snapshot) { s.Handoffs[0].Coverage = nil }, true},
		{"nonfinal reference", func(s *interview.Snapshot) { s.Handoffs[0].EntryIDs = []string{"missing"} }, true},
		{"duplicate handoff", func(s *interview.Snapshot) { s.Handoffs = append(s.Handoffs, s.Handoffs[0]) }, true},
		{"unresolved loss", func(s *interview.Snapshot) { s.Losses[0].Resolved = false }, true},
		{"duplicate entry", func(s *interview.Snapshot) { s.Entries = append(s.Entries, s.Entries[0]) }, true},
		{"excess followups", func(s *interview.Snapshot) {
			s.FollowUps = []string{"background-answer", "background-answer", "background-answer"}
		}, true},
		{"zero time", func(s *interview.Snapshot) { s.StageActivatedAt = time.Time{} }, true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			p := progressFixture(t)
			require.NoError(t, p.RecordTechnicalLoss("loss", "background"))
			answerFixture(t, p)
			require.NoError(t, p.CommitHandoff(enum.StageScreening, instant()))
			require.NoError(t, p.ActivateStage(enum.StageScreening, instant()))
			s := p.Snapshot()
			tt.change(&s)
			_, err := interview.RestoreProgress(s)
			require.ErrorIs(t, err, interview.ErrRestore)
		})
	}
}
