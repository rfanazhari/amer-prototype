package agentconfig_test

import (
	"testing"

	catalog "github.com/rfanazhari/amer-prototype/domain/agentconfig"
	aggregate "github.com/rfanazhari/amer-prototype/domain/aggregate/agentconfig"
	"github.com/rfanazhari/amer-prototype/domain/enum"
	"github.com/rfanazhari/amer-prototype/domain/valueobject"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	jobDescriptionID = "11111111-1111-4111-8111-111111111111"
	salaryCeilingID  = "22222222-2222-4222-8222-222222222222"
)

// hrFindingCase names an HR draft, resolved references, and the expected publication findings.
type hrFindingCase struct {
	name     string
	change   func(*valueobject.HRInput)
	refs     catalog.HRReferenceContext
	eligible bool
	findings []aggregate.Finding
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

// readyHRRefs returns locally ready exact versions for the valid HR example.
func readyHRRefs() catalog.HRReferenceContext {
	return catalog.HRReferenceContext{
		JobDescription: catalog.ReferenceReadiness{Found: true, Ready: true, ID: jobDescriptionID, Version: "1", Role: "Backend Engineer"},
		SalaryCeiling:  catalog.ReferenceReadiness{Found: true, Ready: true, ID: salaryCeilingID, Version: "1"},
	}
}

// mustHR constructs a storage-valid HR draft or fails the test.
func mustHR(t *testing.T, in valueobject.HRInput) valueobject.HRConfiguration {
	t.Helper()
	cfg, err := valueobject.NewHRConfiguration(in)
	require.NoError(t, err)
	return cfg
}

// finding builds an expected detached assessment fact.
func finding(severity enum.FindingSeverity, code, path, message string) aggregate.Finding {
	return aggregate.Finding{Severity: severity, Code: code, Path: path, Message: message}
}

// TestAssessHRCoversPublicationFieldMatrix records errors for incomplete and invalid drafts.
func TestAssessHRCoversPublicationFieldMatrix(t *testing.T) {
	ready := readyHRRefs()
	missingJD := ready
	missingJD.JobDescription = catalog.ReferenceReadiness{}
	unreadyCeiling := ready
	unreadyCeiling.SalaryCeiling.Ready = false
	wrongRole := ready
	wrongRole.JobDescription.Role = "Frontend Engineer"
	tests := []hrFindingCase{
		{name: "valid example", refs: ready, eligible: true},
		{name: "empty draft", change: func(in *valueobject.HRInput) { *in = valueobject.HRInput{} }, eligible: false, findings: []aggregate.Finding{
			finding(enum.FindingError, catalog.CodeRequiredValue, "/jobDescriptionRef", "a required publication value is missing"),
			finding(enum.FindingError, catalog.CodeQuestionStageRequired, "/mandatoryQuestions", "a required interview stage is missing a nonempty question"),
			finding(enum.FindingError, catalog.CodeCompetencyRequired, "/competencyTopics", "at least one competency topic is required"),
			finding(enum.FindingError, catalog.CodeRequiredValue, "/stageSeconds", "a required publication value is missing"),
			finding(enum.FindingError, catalog.CodeRequiredValue, "/compensationPolicy", "a required publication value is missing"),
		}},
		{name: "zero screening budget", change: func(in *valueobject.HRInput) { in.StageSeconds.Screening = 0 }, refs: ready, findings: []aggregate.Finding{
			finding(enum.FindingError, catalog.CodeFollowupBudgetRequired, "/stageSeconds/screening", "screening needs 30 seconds spare for a follow-up"),
			finding(enum.FindingError, catalog.CodeQuestionBudgetExceeded, "/stageSeconds/screening", "stage duration cannot cover the mandatory questions"),
			finding(enum.FindingError, catalog.CodeStageBudgetInvalid, "/stageSeconds/screening", "stage duration must be positive and total 300 to 600 seconds"),
			finding(enum.FindingError, catalog.CodeStageBudgetInvalid, "/stageSeconds", "stage duration must be positive and total 300 to 600 seconds"),
		}},
		{name: "total below 300", change: func(in *valueobject.HRInput) { in.StageSeconds.Screening = 119 }, refs: ready, findings: []aggregate.Finding{
			finding(enum.FindingError, catalog.CodeStageBudgetInvalid, "/stageSeconds", "stage duration must be positive and total 300 to 600 seconds"),
		}},
		{name: "follow-up spare missing", change: func(in *valueobject.HRInput) { in.StageSeconds.Screening = 59 }, refs: ready, findings: []aggregate.Finding{
			finding(enum.FindingError, catalog.CodeFollowupBudgetRequired, "/stageSeconds/screening", "screening needs 30 seconds spare for a follow-up"),
		}},
		{name: "question capacity exceeded", change: func(in *valueobject.HRInput) {
			in.MandatoryQuestions = append(in.MandatoryQuestions, valueobject.QuestionInput{ID: "screen-2", Stage: "screening", Text: "Describe locking."})
			in.StageSeconds.Screening = 50
		}, refs: ready, findings: []aggregate.Finding{
			finding(enum.FindingError, catalog.CodeFollowupBudgetRequired, "/stageSeconds/screening", "screening needs 30 seconds spare for a follow-up"),
			finding(enum.FindingError, catalog.CodeQuestionBudgetExceeded, "/stageSeconds/screening", "stage duration cannot cover the mandatory questions"),
		}},
		{name: "empty question text", change: func(in *valueobject.HRInput) { in.MandatoryQuestions[0].Text = " " }, refs: ready, findings: []aggregate.Finding{
			finding(enum.FindingError, catalog.CodeQuestionTextRequired, "/mandatoryQuestions/0/text", "a mandatory question needs nonempty text"),
			finding(enum.FindingError, catalog.CodeQuestionStageRequired, "/mandatoryQuestions", "a required interview stage is missing a nonempty question"),
		}},
		{name: "job description not ready", refs: missingJD, findings: []aggregate.Finding{
			finding(enum.FindingError, catalog.CodeReferenceNotReady, "/jobDescriptionRef", "a referenced resource is not ready"),
		}},
		{name: "wrong job role", refs: wrongRole, findings: []aggregate.Finding{
			finding(enum.FindingError, catalog.CodeReferenceNotReady, "/jobDescriptionRef", "a referenced resource is not ready"),
		}},
		{name: "salary ceiling not ready", refs: unreadyCeiling, findings: []aggregate.Finding{
			finding(enum.FindingError, catalog.CodeReferenceNotReady, "/compensationPolicy/salaryCeilingRef", "a referenced resource is not ready"),
		}},
		{name: "missing salary ceiling", change: func(in *valueobject.HRInput) { in.CompensationPolicy.SalaryCeilingRef = nil }, refs: ready, findings: []aggregate.Finding{
			finding(enum.FindingError, catalog.CodeRequiredValue, "/compensationPolicy/salaryCeilingRef", "a required publication value is missing"),
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			in := validHRInput()
			if tt.change != nil {
				tt.change(&in)
			}
			findings := catalog.AssessHR(mustHR(t, in), tt.refs)
			assert.Equal(t, tt.findings, findings)
			assert.Equal(t, tt.eligible, catalog.Eligible(findings))
		})
	}
}

// TestEligibleIgnoresWarningsAndBlocksErrors uses independently specified finding sets.
func TestEligibleIgnoresWarningsAndBlocksErrors(t *testing.T) {
	// eligibilityCase names a finding set and the publication decision it must produce.
	type eligibilityCase struct {
		name     string
		findings []aggregate.Finding
		eligible bool
	}
	tests := []eligibilityCase{
		{name: "no findings", eligible: true},
		{name: "warning only", findings: []aggregate.Finding{finding(enum.FindingWarning, catalog.CodeCurrencyNotRequired, "/requiredFields", "total is required without currency")}, eligible: true},
		{name: "error", findings: []aggregate.Finding{finding(enum.FindingError, catalog.CodeRequiredFieldsEmpty, "/requiredFields", "at least one required invoice field must be selected")}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.eligible, catalog.Eligible(tt.findings))
		})
	}
}
