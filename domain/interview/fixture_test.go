package interview_test

import (
	"testing"
	"time"

	"github.com/rfanazhari/amer-prototype/domain/aggregate/agentconfig"
	"github.com/rfanazhari/amer-prototype/domain/enum"
	"github.com/rfanazhari/amer-prototype/domain/interview"
	"github.com/rfanazhari/amer-prototype/domain/valueobject"
	"github.com/stretchr/testify/require"
)

// instant supplies deterministic trusted UTC time.
func instant() time.Time { return time.Date(2026, 11, 2, 0, 0, 0, 123000000, time.UTC) }

// revisionFixture restores a published revision whose schema contains no follow-up cap.
func revisionFixture(t *testing.T) agentconfig.PublishedRevision[valueobject.HRConfiguration] {
	t.Helper()
	c, err := valueobject.NewHRConfiguration(valueobject.HRInput{
		MandatoryQuestions: []valueobject.QuestionInput{{ID: "background", Stage: "background", Text: "Tell us about your work"}, {ID: "screening", Stage: "screening", Text: "Explain a project"}, {ID: "compensation", Stage: "compensation", Text: "What do you expect?"}},
		StageSeconds:       &valueobject.StageSecondsInput{Background: 60, Screening: 240, Compensation: 90, Closing: 30},
		CompensationPolicy: &valueobject.CompensationInput{DiscussionMode: "expectations_only"},
	})
	require.NoError(t, err)
	version, err := agentconfig.NewDraftVersion(1)
	require.NoError(t, err)
	revision, err := agentconfig.RestoreRevision(agentconfig.RevisionSnapshot[valueobject.HRConfiguration]{
		Identity:     agentconfig.RevisionIdentity{ID: "pinned-revision", InstanceID: "instance", Number: 1, DraftVersion: version, Template: agentconfig.TemplateRef{Template: "hr", Version: "1", Behavior: "1"}},
		ValidationID: "publication", Configuration: c, CreatedAt: instant(),
	})
	require.NoError(t, err)
	return revision
}

// progressFixture creates a fresh aggregate using the pinned publication fixture.
func progressFixture(t *testing.T) *interview.Progress {
	t.Helper()
	p, err := interview.NewProgress("execution", revisionFixture(t), instant())
	require.NoError(t, err)
	return p
}

// entryFixture supplies a fresh classified transcript transfer value.
func entryFixture(id string, stage enum.QuestionStage, speaker interview.Speaker, state interview.EvidenceState) interview.TranscriptEntry {
	return interview.TranscriptEntry{ID: id, Stage: stage, Speaker: speaker, State: state, Body: "Recorded speech"}
}

// answerFixture covers the current mandatory question through public behavior.
func answerFixture(t *testing.T, p *interview.Progress) {
	t.Helper()
	stage := p.Snapshot().CurrentStage
	id := stage.String()
	require.NoError(t, p.Ask(id))
	require.NoError(t, p.RecordEntry(entryFixture(id+"-answer", stage, interview.Candidate, interview.Final)))
	require.NoError(t, p.Answer(id, id+"-answer"))
}
