package service_test

import (
	"context"
	"errors"
	"testing"

	"github.com/rfanazhari/amer-prototype/application/service"
	catalog "github.com/rfanazhari/amer-prototype/domain/agentconfig"
	aggregate "github.com/rfanazhari/amer-prototype/domain/aggregate/agentconfig"
	"github.com/rfanazhari/amer-prototype/domain/enum"
	"github.com/rfanazhari/amer-prototype/domain/valueobject"
	"github.com/rfanazhari/amer-prototype/test/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

const (
	jobDescriptionID = "11111111-1111-4111-8111-111111111111"
	salaryCeilingID  = "22222222-2222-4222-8222-222222222222"
)

// assessmentCase names a reference-resolution outcome for HR assessment.
type assessmentCase struct {
	name            string
	jd              service.AgentReferenceFact
	jdFound         bool
	jdErr           error
	ceiling         service.AgentReferenceFact
	ceilingFound    bool
	ceilingErr      error
	wantErr         error
	eligible        bool
	findingCode     string
	wantResolveJD   bool
	wantResolveCeil bool
}

// validHRInput returns the approved valid HR configuration example.
func validHRInput() valueobject.HRInput {
	return valueobject.HRInput{
		JobDescriptionRef: &valueobject.RefInput{ID: jobDescriptionID, Version: "1"},
		MandatoryQuestions: []valueobject.QuestionInput{
			{ID: "background", Stage: "background", Text: "Describe your backend experience."},
			{ID: "screening", Stage: "screening", Text: "How do you protect a concurrent update?"},
			{ID: "compensation", Stage: "compensation", Text: "What are your compensation expectations?"},
		},
		CompetencyTopics: []string{"concurrency"},
		StageSeconds:     &valueobject.StageSecondsInput{Background: 60, Screening: 240, Compensation: 90, Closing: 30},
		CompensationPolicy: &valueobject.CompensationInput{
			DiscussionMode:   "expectations_only",
			SalaryCeilingRef: &valueobject.RefInput{ID: salaryCeilingID, Version: "1"},
		},
	}
}

// mustHR constructs a storage-valid HR draft or fails the test.
func mustHR(t *testing.T, in valueobject.HRInput) valueobject.HRConfiguration {
	t.Helper()
	cfg, err := valueobject.NewHRConfiguration(in)
	require.NoError(t, err)
	return cfg
}

// hrTemplate is the catalog HR Interviewer v1 reference.
func hrTemplate() aggregate.TemplateRef {
	return aggregate.TemplateRef{Template: catalog.TemplateHRInterviewer, Version: catalog.TemplateVersion1, Behavior: catalog.BehaviorVersion1}
}

// documentTemplate is the catalog Document Analyzer v1 reference.
func documentTemplate() aggregate.TemplateRef {
	return aggregate.TemplateRef{Template: catalog.TemplateDocumentAnalyzer, Version: catalog.TemplateVersion1, Behavior: catalog.BehaviorVersion1}
}

// TestDraftAssessmentCatalogListsBothTemplates does not call reference or model ports.
func TestDraftAssessmentCatalogListsBothTemplates(t *testing.T) {
	reader := mocks.NewAgentReferenceReader[service.AgentReferenceFact](t)
	got := (service.DraftAssessment{References: reader}).Catalog()
	require.Len(t, got, 2)
	assert.Equal(t, catalog.TemplateHRInterviewer, got[0].ID)
	assert.Equal(t, catalog.TemplateDocumentAnalyzer, got[1].ID)
	assert.Equal(t, catalog.TemplateVersion1, got[0].Version)
	assert.Equal(t, catalog.BehaviorVersion1, got[0].Behavior)
}

// TestAssessHRPreservesExactReferenceVersionsAndClassifiesReadiness uses a generated reader mock.
func TestAssessHRPreservesExactReferenceVersionsAndClassifiesReadiness(t *testing.T) {
	readyJD := service.AgentReferenceFact{Kind: service.ReferenceKindJobDescription, ID: jobDescriptionID, Version: "1", Role: service.DemoJobRole, Ready: true}
	readyCeiling := service.AgentReferenceFact{Kind: service.ReferenceKindCompensationPolicy, ID: salaryCeilingID, Version: "1", Ready: true}
	unavailable := errors.New("provider unavailable")
	tests := []assessmentCase{
		{name: "ready exact versions", jd: readyJD, jdFound: true, ceiling: readyCeiling, ceilingFound: true, eligible: true, wantResolveJD: true, wantResolveCeil: true},
		{name: "missing job description", ceiling: readyCeiling, ceilingFound: true, findingCode: catalog.CodeReferenceNotReady, wantResolveJD: true, wantResolveCeil: true},
		{name: "newer job description identity", jd: service.AgentReferenceFact{Kind: service.ReferenceKindJobDescription, ID: jobDescriptionID, Version: "2", Role: service.DemoJobRole, Ready: true}, jdFound: true, wantErr: service.ErrReferenceIdentity, wantResolveJD: true},
		{name: "reader unavailable", jdErr: unavailable, wantErr: unavailable, wantResolveJD: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := mocks.NewAgentReferenceReader[service.AgentReferenceFact](t)
			if tt.wantResolveJD {
				reader.On("Resolve", mock.Anything, service.AgentReference{Kind: service.ReferenceKindJobDescription, ID: jobDescriptionID, Version: "1"}).
					Return(tt.jd, tt.jdFound, tt.jdErr).Once()
			}
			if tt.wantResolveCeil {
				reader.On("Resolve", mock.Anything, service.AgentReference{Kind: service.ReferenceKindCompensationPolicy, ID: salaryCeilingID, Version: "1"}).
					Return(tt.ceiling, tt.ceilingFound, tt.ceilingErr).Once()
			}
			got, err := (service.DraftAssessment{References: reader}).AssessHR(context.Background(), hrTemplate(), mustHR(t, validHRInput()))
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				assert.Equal(t, service.DraftAssessmentResult{}, got)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, catalog.TemplateHRInterviewer, got.TemplateID)
			assert.Equal(t, catalog.TemplateVersion1, got.TemplateVersion)
			assert.Equal(t, catalog.BehaviorVersion1, got.BehaviorVersion)
			assert.Equal(t, tt.eligible, got.Eligible)
			if tt.findingCode == "" {
				assert.Empty(t, got.Findings)
				return
			}
			require.NotEmpty(t, got.Findings)
			assert.Equal(t, tt.findingCode, got.Findings[0].Code)
			assert.False(t, got.Eligible)
		})
	}
}

// TestAssessHRRejectsUnknownTemplate does not resolve references for unsupported identities.
func TestAssessHRRejectsUnknownTemplate(t *testing.T) {
	reader := mocks.NewAgentReferenceReader[service.AgentReferenceFact](t)
	_, err := (service.DraftAssessment{References: reader}).AssessHR(context.Background(), aggregate.TemplateRef{Template: "custom-agent", Version: "1", Behavior: "1"}, mustHR(t, validHRInput()))
	require.ErrorIs(t, err, catalog.ErrUnknownTemplate)
}

// TestAssessDocumentDoesNotCallReferenceReader proves local validation has no provider side effects.
func TestAssessDocumentDoesNotCallReferenceReader(t *testing.T) {
	reader := mocks.NewAgentReferenceReader[service.AgentReferenceFact](t)
	cfg, err := valueobject.NewDocumentConfiguration([]string{"total"})
	require.NoError(t, err)
	got, err := (service.DraftAssessment{References: reader}).AssessDocument(context.Background(), documentTemplate(), cfg)
	require.NoError(t, err)
	assert.True(t, got.Eligible)
	require.Len(t, got.Findings, 1)
	assert.Equal(t, catalog.CodeCurrencyNotRequired, got.Findings[0].Code)
	assert.Equal(t, enum.FindingWarning, got.Findings[0].Severity)
	assert.Equal(t, catalog.TemplateDocumentAnalyzer, got.TemplateID)
}
