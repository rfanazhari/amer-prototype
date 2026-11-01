package agentconfig_test

import (
	"testing"

	catalog "github.com/rfanazhari/amer-prototype/domain/agentconfig"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// catalogLookupCase names a template identity that must be found or rejected.
type catalogLookupCase struct {
	name, id, version string
	wantFound         bool
	wantID            string
	wantBehavior      string
	wantDisplay       string
}

// TestTemplatesDiscoverBothVersionedEntries exposes HR Interviewer and Document Analyzer v1.
func TestTemplatesDiscoverBothVersionedEntries(t *testing.T) {
	got := catalog.Templates()
	require.Len(t, got, 2)
	assert.Equal(t, catalog.Template{
		ID: catalog.TemplateHRInterviewer, Version: catalog.TemplateVersion1, Behavior: catalog.BehaviorVersion1,
		DisplayName: catalog.DisplayHRInterviewer, WarningPolicy: catalog.WarningPolicyInformational,
	}, got[0])
	assert.Equal(t, catalog.Template{
		ID: catalog.TemplateDocumentAnalyzer, Version: catalog.TemplateVersion1, Behavior: catalog.BehaviorVersion1,
		DisplayName: catalog.DisplayDocumentAnalyzer, WarningPolicy: catalog.WarningPolicyInformational,
	}, got[1])
	assert.Equal(t, catalog.TemplateHRInterviewer, got[0].Ref().Template)
	assert.Equal(t, catalog.BehaviorVersion1, got[0].Ref().Behavior)
}

// TestLookupSelectsExactTemplateVersions rejects unknown identities without guessing latest.
func TestLookupSelectsExactTemplateVersions(t *testing.T) {
	tests := []catalogLookupCase{
		{name: "hr v1", id: catalog.TemplateHRInterviewer, version: "1", wantFound: true, wantID: catalog.TemplateHRInterviewer, wantBehavior: "1", wantDisplay: catalog.DisplayHRInterviewer},
		{name: "document v1", id: catalog.TemplateDocumentAnalyzer, version: "1", wantFound: true, wantID: catalog.TemplateDocumentAnalyzer, wantBehavior: "1", wantDisplay: catalog.DisplayDocumentAnalyzer},
		{name: "unknown template", id: "custom-agent", version: "1"},
		{name: "unknown hr version", id: catalog.TemplateHRInterviewer, version: "2"},
		{name: "empty", id: "", version: "1"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := catalog.Lookup(tt.id, tt.version)
			assert.Equal(t, tt.wantFound, ok)
			if !tt.wantFound {
				assert.Equal(t, catalog.Template{}, got)
				return
			}
			assert.Equal(t, tt.wantID, got.ID)
			assert.Equal(t, tt.version, got.Version)
			assert.Equal(t, tt.wantBehavior, got.Behavior)
			assert.Equal(t, tt.wantDisplay, got.DisplayName)
			assert.Equal(t, catalog.WarningPolicyInformational, got.WarningPolicy)
		})
	}
}
