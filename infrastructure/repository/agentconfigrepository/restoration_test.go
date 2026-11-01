package agentconfigrepository

import (
	"testing"

	catalog "github.com/rfanazhari/amer-prototype/domain/agentconfig"
	"github.com/rfanazhari/amer-prototype/domain/valueobject"
	"github.com/rfanazhari/amer-prototype/test/testdata"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// historicalHRRevisionJSON is a stored v1 revision that would fail current publication assessment.
const historicalHRRevisionJSON = `{"jobDescriptionRef":{"id":"11111111-1111-4111-8111-111111111111","version":"1"},"mandatoryQuestions":[{"id":"background","stage":"background","text":"Historical single-question revision."}],"competencyTopics":[],"stageSeconds":null,"compensationPolicy":null}`

// TestHistoricalRevisionRestoresWithoutCurrentAssessor proves restoration keeps stored shape without AssessHR.
func TestHistoricalRevisionRestoresWithoutCurrentAssessor(t *testing.T) {
	model := RevisionModel{
		ID: "cccccccc-cccc-4ccc-8ccc-ccccccccccc8", InstanceID: "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaa8",
		RevisionNumber: 1, SourceDraftVersion: 1, ValidationID: "bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbb8",
		TemplateID: catalog.TemplateHRInterviewer, TemplateVersion: catalog.TemplateVersion1, BehaviorVersion: catalog.BehaviorVersion1,
		Configuration: JSONDocument(historicalHRRevisionJSON), CreatedAt: testdata.Now(),
	}
	stored, err := toStoredRevision(model)
	require.NoError(t, err)
	require.NotNil(t, stored.HR)
	snap := stored.HR.Snapshot()
	findings := catalog.AssessHR(snap.Configuration, catalog.HRReferenceContext{
		JobDescription: catalog.ReferenceReadiness{Found: true, Ready: true, ID: "11111111-1111-4111-8111-111111111111", Version: "1", Role: "Backend Engineer"},
	})
	assert.False(t, catalog.Eligible(findings))
	assert.Len(t, snap.Configuration.MandatoryQuestions(), 1)
	assert.Equal(t, "Historical single-question revision.", snap.Configuration.MandatoryQuestions()[0].Text())
}

// TestStoredRevisionRejectsUnsupportedBehaviorVersion prevents loading identities outside restoration support.
func TestStoredRevisionRejectsUnsupportedBehaviorVersion(t *testing.T) {
	cfg, err := valueobject.NewHRConfiguration(valueobject.HRInput{})
	require.NoError(t, err)
	body, err := encodeHRConfiguration(cfg)
	require.NoError(t, err)
	model := RevisionModel{
		ID: "cccccccc-cccc-4ccc-8ccc-ccccccccccc8", InstanceID: "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaa8",
		RevisionNumber: 1, SourceDraftVersion: 1, ValidationID: "bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbb8",
		TemplateID: catalog.TemplateHRInterviewer, TemplateVersion: catalog.TemplateVersion1, BehaviorVersion: "2",
		Configuration: body, CreatedAt: testdata.Now(),
	}
	_, err = toStoredRevision(model)
	require.Error(t, err)
	assert.ErrorIs(t, err, catalog.ErrUnsupportedStoredTemplate)
}
