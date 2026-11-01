package dto

import "time"

// AgentTemplate is a detached catalog entry without provider or schema-engine details.
type AgentTemplate struct {
	TemplateID, TemplateVersion, BehaviorVersion, DisplayName, WarningPolicy string
}

// AgentInstance is a detached instance view with confidentiality-filtered configuration.
type AgentInstance struct {
	ID, Name, TemplateID, TemplateVersion, BehaviorVersion string
	DraftVersion                                           uint64
	Configuration                                          AgentConfigurationView
	ActiveRevisionID                                       *string
	CreatedAt, UpdatedAt                                   time.Time
}

// AgentValidation is a detached exact-version assessment result.
type AgentValidation struct {
	ValidationID, InstanceID, TemplateID, TemplateVersion, BehaviorVersion string
	DraftVersion                                                           uint64
	Eligible                                                               bool
	Findings                                                               []AgentFinding
	CreatedAt                                                              time.Time
}

// AgentFinding is a detached validation finding without echoed rejected values.
type AgentFinding struct {
	Severity, Code, Path, Message string
}

// AgentRevision is a detached immutable revision view with confidentiality filtering.
type AgentRevision struct {
	RevisionID, InstanceID, ValidationID, TemplateID, TemplateVersion, BehaviorVersion string
	RevisionNumber, SourceDraftVersion                                                 uint64
	Configuration                                                                      AgentConfigurationView
	PublishedAt                                                                        time.Time
}

// AgentConfigurationView is a template-tagged filtered configuration body.
type AgentConfigurationView struct {
	HR       *HRConfigurationView
	Document *DocumentConfigurationView
}

// HRConfigurationView is the authorized HR draft or revision body without salary-reference contents.
type HRConfigurationView struct {
	JobDescriptionRef  *StableRefView
	MandatoryQuestions []MandatoryQuestionView
	CompetencyTopics   []string
	StageSeconds       *StageSecondsView
	CompensationPolicy *CompensationPolicyView
}

// StableRefView is a detached immutable reference identity.
type StableRefView struct {
	ID, Version string
}

// MandatoryQuestionView is a detached mandatory-question body.
type MandatoryQuestionView struct {
	ID, Stage, Text string
}

// StageSecondsView is a detached four-stage duration object.
type StageSecondsView struct {
	Background, Screening, Compensation, Closing int
}

// CompensationPolicyView exposes discussion mode and whether a ceiling is configured, never the reference.
type CompensationPolicyView struct {
	DiscussionMode          string
	SalaryCeilingConfigured bool
}

// DocumentConfigurationView is a detached Document Analyzer required-field list in catalog order.
type DocumentConfigurationView struct {
	RequiredFields []string
}
