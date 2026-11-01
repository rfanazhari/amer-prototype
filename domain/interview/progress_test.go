package interview_test

import (
	"testing"
	"time"

	"github.com/rfanazhari/amer-prototype/domain/aggregate/agentconfig"
	"github.com/rfanazhari/amer-prototype/domain/enum"
	"github.com/rfanazhari/amer-prototype/domain/interview"
	"github.com/rfanazhari/amer-prototype/domain/valueobject"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestProgressCreation proves pinned questions, published seconds and frozen policy.
func TestProgressCreation(t *testing.T) {
	// creationCase selects valid and invalid construction inputs.
	type creationCase struct {
		name            string
		id              string
		missingRevision bool
		at              time.Time
		wantErr         bool
	}
	for _, tt := range []creationCase{
		{"pinned", "execution", false, instant().In(time.FixedZone("caller", 7*3600)).Add(321 * time.Nanosecond), false},
		{"missing execution", "", false, instant(), true},
		{"missing revision", "execution", true, instant(), true},
		{"missing time", "execution", false, time.Time{}, true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			revision := revisionFixture(t)
			if tt.missingRevision {
				revision = agentconfig.PublishedRevision[valueobject.HRConfiguration]{}
			}
			p, err := interview.NewProgress(tt.id, revision, tt.at)
			if tt.wantErr {
				require.ErrorIs(t, err, interview.ErrInvalid)
				return
			}
			require.NoError(t, err)
			s := p.Snapshot()
			assert.Equal(t, "execution", s.ExecutionID)
			assert.Equal(t, "pinned-revision", s.RevisionID)
			assert.Equal(t, 2, s.FollowUpLimit)
			assert.Equal(t, enum.StageBackground, s.CurrentStage)
			assert.Equal(t, instant(), s.RunningAt)
			assert.Equal(t, instant(), s.StageActivatedAt)
			assert.Equal(t, 60, s.Configuration.StageSeconds().Background())
			require.Len(t, s.Coverage, 3)
			for _, item := range s.Coverage {
				assert.Equal(t, interview.Pending, item.State)
			}
			assert.False(t, p.ReadyForCompletion())
		})
	}
}

// TestCoverageEdges proves legal transitions, terminal immutability and failure atomicity.
func TestCoverageEdges(t *testing.T) {
	// edgeCase chooses the source coverage and the attempted action.
	type edgeCase struct {
		name    string
		source  interview.CoverageState
		action  string
		wantErr bool
		want    interview.CoverageState
	}
	for _, tt := range []edgeCase{
		{"pending ask", interview.Pending, "ask", false, interview.Asked},
		{"pending answer", interview.Pending, "answer", true, interview.Pending},
		{"asked ask", interview.Asked, "ask", true, interview.Asked},
		{"asked answer", interview.Asked, "answer", false, interview.Answered},
		{"answered ask", interview.Answered, "ask", true, interview.Answered},
		{"answered answer", interview.Answered, "answer", true, interview.Answered},
		{"answered omit", interview.Answered, "omit", true, interview.Answered},
		{"omitted ask", interview.Omitted, "ask", true, interview.Omitted},
		{"omitted answer", interview.Omitted, "answer", true, interview.Omitted},
		{"omitted omit", interview.Omitted, "omit", true, interview.Omitted},
	} {
		t.Run(tt.name, func(t *testing.T) {
			p := progressFixture(t)
			require.NoError(t, p.RecordEntry(entryFixture("answer", enum.StageBackground, interview.Candidate, interview.Final)))
			if tt.source == interview.Asked || tt.source == interview.Answered {
				require.NoError(t, p.Ask("background"))
			}
			if tt.source == interview.Answered {
				require.NoError(t, p.Answer("background", "answer"))
			}
			if tt.source == interview.Omitted {
				require.NoError(t, p.Omit("background", interview.SoftBudgetExpired, "", instant().Add(time.Minute)))
			}
			before := p.Snapshot()
			var err error
			switch tt.action {
			case "ask":
				err = p.Ask("background")
			case "answer":
				err = p.Answer("background", "answer")
			case "omit":
				err = p.Omit("background", interview.SoftBudgetExpired, "", instant().Add(time.Minute))
			}
			if tt.wantErr {
				require.Error(t, err)
				assert.Equal(t, before, p.Snapshot())
			} else {
				require.NoError(t, err)
			}
			assert.Equal(t, tt.want, p.Snapshot().Coverage[0].State)
		})
	}
}

// TestCitations proves fragments, model speech and foreign stages are not answers or follow-ups.
func TestCitations(t *testing.T) {
	// citationCase describes the classified evidence supplied by the caller.
	type citationCase struct {
		name    string
		speaker interview.Speaker
		state   interview.EvidenceState
		missing bool
		wantErr bool
	}
	for _, tt := range []citationCase{
		{"final candidate", interview.Candidate, interview.Final, false, false},
		{"fragment", interview.Candidate, interview.Fragment, false, true},
		{"model", interview.Model, interview.Final, false, true},
		{"missing", interview.Candidate, interview.Final, true, true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			p := progressFixture(t)
			require.NoError(t, p.Ask("background"))
			if !tt.missing {
				require.NoError(t, p.RecordEntry(entryFixture("evidence", enum.StageBackground, tt.speaker, tt.state)))
			}
			before := p.Snapshot()
			err := p.Answer("background", "evidence")
			if tt.wantErr {
				require.ErrorIs(t, err, interview.ErrEvidence)
				assert.Equal(t, before, p.Snapshot())
				require.ErrorIs(t, p.FollowUp("evidence"), interview.ErrEvidence)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, "evidence", p.Snapshot().Coverage[0].AnswerEntryID)
			require.NoError(t, p.FollowUp("evidence"))
			require.NoError(t, p.FollowUp("evidence"))
			before = p.Snapshot()
			require.ErrorIs(t, p.FollowUp("evidence"), interview.ErrBudget)
			assert.Equal(t, before, p.Snapshot())
			require.NoError(t, p.CommitHandoff(enum.StageScreening, instant()))
			require.NoError(t, p.ActivateStage(enum.StageScreening, instant()))
			require.ErrorIs(t, p.FollowUp("evidence"), interview.ErrEvidence)
			require.NoError(t, p.Ask("screening"))
			require.ErrorIs(t, p.Answer("screening", "evidence"), interview.ErrEvidence)
		})
	}
}

// TestOmissionReasons proves each approved reason and its finalized prerequisites.
func TestOmissionReasons(t *testing.T) {
	// omissionCase selects reason, evidence and trusted time.
	type omissionCase struct {
		name               string
		reason             interview.OmissionReason
		ask, clarify, skip bool
		at                 time.Time
		wantErr            bool
	}
	for _, tt := range []omissionCase{
		{"skip after clarification", interview.CandidateSkip, true, true, true, instant(), false},
		{"skip without clarification", interview.CandidateSkip, true, false, true, instant(), true},
		{"skip without final observation", interview.CandidateSkip, true, true, false, instant(), true},
		{"silence after clarification", interview.NoAnswerAfterClarification, true, true, false, instant(), false},
		{"silence without clarification", interview.NoAnswerAfterClarification, true, false, false, instant(), true},
		{"pending at budget", interview.SoftBudgetExpired, false, false, false, instant().Add(time.Minute), false},
		{"asked at budget", interview.SoftBudgetExpired, true, false, false, instant().Add(time.Minute), false},
		{"before budget", interview.SoftBudgetExpired, true, false, false, instant().Add(time.Minute - time.Millisecond), true},
		{"technical is not omission", "technical_loss", true, true, true, instant(), true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			p := progressFixture(t)
			if tt.ask {
				require.NoError(t, p.Ask("background"))
			}
			if tt.clarify {
				require.NoError(t, p.RecordEntry(entryFixture("clarify", enum.StageBackground, interview.Model, interview.Final)))
				require.NoError(t, p.Clarify("background", "clarify"))
				before := p.Snapshot()
				require.Error(t, p.Clarify("background", "clarify"))
				assert.Equal(t, before, p.Snapshot())
			}
			citation := ""
			if tt.skip {
				citation = "skip"
				require.NoError(t, p.RecordEntry(entryFixture(citation, enum.StageBackground, interview.Candidate, interview.Final)))
			}
			before := p.Snapshot()
			err := p.Omit("background", tt.reason, citation, tt.at)
			if tt.wantErr {
				require.Error(t, err)
				assert.Equal(t, before, p.Snapshot())
				return
			}
			require.NoError(t, err)
			assert.Equal(t, interview.Omitted, p.Snapshot().Coverage[0].State)
			assert.Equal(t, tt.reason, p.Snapshot().Coverage[0].OmissionReason)
		})
	}
}

// TestTechnicalLoss proves technical gaps preserve coverage and cannot authorize progression.
func TestTechnicalLoss(t *testing.T) {
	// lossCase selects whether speech had been asked before the fault.
	type lossCase struct {
		name    string
		asked   bool
		wantErr bool
	}
	for _, tt := range []lossCase{{"pending loss", false, false}, {"asked loss", true, false}} {
		t.Run(tt.name, func(t *testing.T) {
			p := progressFixture(t)
			if tt.asked {
				require.NoError(t, p.Ask("background"))
			}
			state := p.Snapshot().Coverage[0].State
			require.NoError(t, p.RecordTechnicalLoss("loss", "background"))
			assert.Equal(t, state, p.Snapshot().Coverage[0].State)
			assert.Empty(t, p.Snapshot().Coverage[0].OmissionReason)
			before := p.Snapshot()
			require.Error(t, p.Omit("background", interview.SoftBudgetExpired, "", instant().Add(time.Minute)))
			require.Error(t, p.Authorize(enum.StageScreening, instant().Add(time.Minute)))
			assert.Equal(t, before, p.Snapshot())
			if !tt.asked {
				require.NoError(t, p.Ask("background"))
			}
			require.NoError(t, p.RecordEntry(entryFixture("recovered", enum.StageBackground, interview.Candidate, interview.Final)))
			require.NoError(t, p.Answer("background", "recovered"))
			assert.True(t, p.Snapshot().Losses[0].Resolved)
			require.NoError(t, p.Authorize(enum.StageScreening, instant().Add(time.Minute)))
		})
	}
}
