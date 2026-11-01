package agentconfig

import (
	"sort"

	aggregate "github.com/rfanazhari/amer-prototype/domain/aggregate/agentconfig"
	"github.com/rfanazhari/amer-prototype/domain/enum"
)

const (
	CodeRequiredValue          = "REQUIRED_VALUE"
	CodeQuestionTextRequired   = "QUESTION_TEXT_REQUIRED"
	CodeQuestionStageRequired  = "QUESTION_STAGE_REQUIRED"
	CodeCompetencyRequired     = "COMPETENCY_REQUIRED"
	CodeStageBudgetInvalid     = "STAGE_BUDGET_INVALID"
	CodeQuestionBudgetExceeded = "QUESTION_BUDGET_EXCEEDED"
	CodeFollowupBudgetRequired = "FOLLOWUP_BUDGET_REQUIRED"
	CodeReferenceNotReady      = "REFERENCE_NOT_READY"
	CodeRequiredFieldsEmpty    = "REQUIRED_FIELDS_EMPTY"
	CodeCurrencyNotRequired    = "CURRENCY_NOT_REQUIRED"
)

const (
	messageRequiredValue          = "a required publication value is missing"
	messageQuestionTextRequired   = "a mandatory question needs nonempty text"
	messageQuestionStageRequired  = "a required interview stage is missing a nonempty question"
	messageCompetencyRequired     = "at least one competency topic is required"
	messageStageBudgetInvalid     = "stage duration must be positive and total 300 to 600 seconds"
	messageQuestionBudgetExceeded = "stage duration cannot cover the mandatory questions"
	messageFollowupBudgetRequired = "screening needs 30 seconds spare for a follow-up"
	messageReferenceNotReady      = "a referenced resource is not ready"
	messageRequiredFieldsEmpty    = "at least one required invoice field must be selected"
	messageCurrencyNotRequired    = "total is required without currency"
)

// Eligible reports whether findings contain no publication-blocking errors.
func Eligible(findings []aggregate.Finding) bool {
	for _, finding := range findings {
		if finding.Severity == enum.FindingError {
			return false
		}
	}
	return true
}

// FindingCodes returns the stable validation finding codes that require EN/ID messages.
func FindingCodes() []string {
	return []string{
		CodeRequiredValue, CodeQuestionTextRequired, CodeQuestionStageRequired, CodeCompetencyRequired,
		CodeStageBudgetInvalid, CodeQuestionBudgetExceeded, CodeFollowupBudgetRequired, CodeReferenceNotReady,
		CodeRequiredFieldsEmpty, CodeCurrencyNotRequired,
	}
}

// errorFinding constructs a publication-blocking assessment fact.
func errorFinding(code, path, message string) aggregate.Finding {
	return aggregate.Finding{Severity: enum.FindingError, Code: code, Path: path, Message: message}
}

// warningFinding constructs an informational assessment fact that does not block publication.
func warningFinding(code, path, message string) aggregate.Finding {
	return aggregate.Finding{Severity: enum.FindingWarning, Code: code, Path: path, Message: message}
}

// sortFindings orders by path, error-before-warning severity, then code.
func sortFindings(findings []aggregate.Finding) {
	sort.SliceStable(findings, func(i, j int) bool {
		a, b := findings[i], findings[j]
		if a.Path != b.Path {
			return a.Path < b.Path
		}
		if a.Severity != b.Severity {
			return a.Severity == enum.FindingError
		}
		return a.Code < b.Code
	})
}
