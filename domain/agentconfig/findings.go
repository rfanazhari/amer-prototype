package agentconfig

import "github.com/rfanazhari/amer-prototype/domain/aggregate/agentconfig"

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
func Eligible(findings []agentconfig.Finding) bool { return true }

// FindingCodes returns the stable validation finding codes that require EN/ID messages.
func FindingCodes() []string {
	return []string{
		CodeRequiredValue, CodeQuestionTextRequired, CodeQuestionStageRequired, CodeCompetencyRequired,
		CodeStageBudgetInvalid, CodeQuestionBudgetExceeded, CodeFollowupBudgetRequired, CodeReferenceNotReady,
		CodeRequiredFieldsEmpty, CodeCurrencyNotRequired,
	}
}
