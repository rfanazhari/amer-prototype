package agentconfighandler

import (
	"time"

	"github.com/rfanazhari/amer-prototype/application/dto"
	catalog "github.com/rfanazhari/amer-prototype/domain/agentconfig"
)

// catalogBody is the catalog success payload.
type catalogBody struct {
	Templates []templateBody `json:"templates"`
}

// templateBody is one catalog entry including typed configuration constraints.
type templateBody struct {
	TemplateID          string `json:"templateId"`
	TemplateVersion     string `json:"templateVersion"`
	BehaviorVersion     string `json:"behaviorVersion"`
	DisplayName         string `json:"displayName"`
	ConfigurationSchema any    `json:"configurationSchema"`
	WarningPolicy       string `json:"warningPolicy"`
}

// instanceBody is the instance success payload with a template-shaped configuration object.
type instanceBody struct {
	ID               string  `json:"id"`
	Name             string  `json:"name"`
	TemplateID       string  `json:"templateId"`
	TemplateVersion  string  `json:"templateVersion"`
	BehaviorVersion  string  `json:"behaviorVersion"`
	DraftVersion     uint64  `json:"draftVersion"`
	Configuration    any     `json:"configuration"`
	ActiveRevisionID *string `json:"activeRevisionId"`
	CreatedAt        string  `json:"createdAt"`
	UpdatedAt        string  `json:"updatedAt"`
}

// validationBody is the persisted assessment payload, including blocking findings.
type validationBody struct {
	ValidationID    string        `json:"validationId"`
	InstanceID      string        `json:"instanceId"`
	DraftVersion    uint64        `json:"draftVersion"`
	TemplateID      string        `json:"templateId"`
	TemplateVersion string        `json:"templateVersion"`
	BehaviorVersion string        `json:"behaviorVersion"`
	Eligible        bool          `json:"eligible"`
	Findings        []findingBody `json:"findings"`
	CreatedAt       string        `json:"createdAt"`
}

// findingBody is one localized assessment finding without echoed rejected values.
type findingBody struct {
	Severity string `json:"severity"`
	Code     string `json:"code"`
	Path     string `json:"path"`
	Message  string `json:"message"`
}

// revisionBody is the immutable revision success payload.
type revisionBody struct {
	RevisionID         string `json:"revisionId"`
	InstanceID         string `json:"instanceId"`
	RevisionNumber     uint64 `json:"revisionNumber"`
	SourceDraftVersion uint64 `json:"sourceDraftVersion"`
	ValidationID       string `json:"validationId"`
	TemplateID         string `json:"templateId"`
	TemplateVersion    string `json:"templateVersion"`
	BehaviorVersion    string `json:"behaviorVersion"`
	Configuration      any    `json:"configuration"`
	PublishedAt        string `json:"publishedAt"`
}

// hrBody is the confidentiality-filtered HR configuration object.
type hrBody struct {
	JobDescriptionRef  *refBody          `json:"jobDescriptionRef"`
	MandatoryQuestions []questionBody    `json:"mandatoryQuestions"`
	CompetencyTopics   []string          `json:"competencyTopics"`
	StageSeconds       *stageBody        `json:"stageSeconds"`
	CompensationPolicy *compensationBody `json:"compensationPolicy"`
}

// documentBody is the Document Analyzer configuration object.
type documentBody struct {
	RequiredFields []string `json:"requiredFields"`
}

// refBody is a stable reference identity.
type refBody struct {
	ID      string `json:"id"`
	Version string `json:"version"`
}

// questionBody is one mandatory question.
type questionBody struct {
	ID    string `json:"id"`
	Stage string `json:"stage"`
	Text  string `json:"text"`
}

// stageBody is the four-stage duration object.
type stageBody struct {
	Background   int `json:"background"`
	Screening    int `json:"screening"`
	Compensation int `json:"compensation"`
	Closing      int `json:"closing"`
}

// compensationBody exposes discussion mode and whether a ceiling is configured.
type compensationBody struct {
	DiscussionMode          string `json:"discussionMode"`
	SalaryCeilingConfigured bool   `json:"salaryCeilingConfigured"`
}

// formatted renders a UTC instant with exactly three fractional digits.
func formatted(v time.Time) string {
	return v.UTC().Format("2006-01-02T15:04:05.000Z")
}

// encodeCatalog copies detached catalog identities onto the public schema payload.
func encodeCatalog(templates []dto.AgentTemplate) catalogBody {
	out := make([]templateBody, len(templates))
	for i, template := range templates {
		out[i] = templateBody{
			TemplateID: template.TemplateID, TemplateVersion: template.TemplateVersion, BehaviorVersion: template.BehaviorVersion,
			DisplayName: template.DisplayName, WarningPolicy: template.WarningPolicy, ConfigurationSchema: configurationSchema(template.TemplateID),
		}
	}
	return catalogBody{Templates: out}
}

// encodeInstance copies a detached instance, using null for an absent active revision.
func encodeInstance(instance *dto.AgentInstance) instanceBody {
	return instanceBody{
		ID: instance.ID, Name: instance.Name, TemplateID: instance.TemplateID, TemplateVersion: instance.TemplateVersion,
		BehaviorVersion: instance.BehaviorVersion, DraftVersion: instance.DraftVersion, Configuration: encodeConfiguration(instance.Configuration),
		ActiveRevisionID: instance.ActiveRevisionID, CreatedAt: formatted(instance.CreatedAt), UpdatedAt: formatted(instance.UpdatedAt),
	}
}

// encodeValidation copies a detached assessment and keeps findings a non-null array.
func encodeValidation(result *dto.AgentValidation) validationBody {
	findings := make([]findingBody, 0, len(result.Findings))
	for _, finding := range result.Findings {
		findings = append(findings, findingBody{Severity: finding.Severity, Code: finding.Code, Path: finding.Path, Message: finding.Message})
	}
	return validationBody{
		ValidationID: result.ValidationID, InstanceID: result.InstanceID, DraftVersion: result.DraftVersion,
		TemplateID: result.TemplateID, TemplateVersion: result.TemplateVersion, BehaviorVersion: result.BehaviorVersion,
		Eligible: result.Eligible, Findings: findings, CreatedAt: formatted(result.CreatedAt),
	}
}

// encodeRevision copies a detached immutable revision.
func encodeRevision(revision *dto.AgentRevision) revisionBody {
	return revisionBody{
		RevisionID: revision.RevisionID, InstanceID: revision.InstanceID, RevisionNumber: revision.RevisionNumber,
		SourceDraftVersion: revision.SourceDraftVersion, ValidationID: revision.ValidationID, TemplateID: revision.TemplateID,
		TemplateVersion: revision.TemplateVersion, BehaviorVersion: revision.BehaviorVersion, Configuration: encodeConfiguration(revision.Configuration),
		PublishedAt: formatted(revision.PublishedAt),
	}
}

// encodeConfiguration selects the template-shaped configuration object.
func encodeConfiguration(view dto.AgentConfigurationView) any {
	if view.Document != nil {
		fields := view.Document.RequiredFields
		if fields == nil {
			fields = []string{}
		}
		return documentBody{RequiredFields: fields}
	}
	if view.HR == nil {
		return hrBody{MandatoryQuestions: []questionBody{}, CompetencyTopics: []string{}}
	}
	questions := make([]questionBody, 0, len(view.HR.MandatoryQuestions))
	for _, question := range view.HR.MandatoryQuestions {
		questions = append(questions, questionBody{ID: question.ID, Stage: question.Stage, Text: question.Text})
	}
	topics := view.HR.CompetencyTopics
	if topics == nil {
		topics = []string{}
	}
	body := hrBody{MandatoryQuestions: questions, CompetencyTopics: topics}
	if view.HR.JobDescriptionRef != nil {
		body.JobDescriptionRef = &refBody{ID: view.HR.JobDescriptionRef.ID, Version: view.HR.JobDescriptionRef.Version}
	}
	if view.HR.StageSeconds != nil {
		body.StageSeconds = &stageBody{Background: view.HR.StageSeconds.Background, Screening: view.HR.StageSeconds.Screening, Compensation: view.HR.StageSeconds.Compensation, Closing: view.HR.StageSeconds.Closing}
	}
	if view.HR.CompensationPolicy != nil {
		body.CompensationPolicy = &compensationBody{DiscussionMode: view.HR.CompensationPolicy.DiscussionMode, SalaryCeilingConfigured: view.HR.CompensationPolicy.SalaryCeilingConfigured}
	}
	return body
}

// configurationSchema returns the approved field constraints for a catalog template.
func configurationSchema(templateID string) any {
	if templateID == catalog.TemplateDocumentAnalyzer {
		return documentConfigurationSchema()
	}
	return hrConfigurationSchema()
}
