package dto

import (
	catalog "github.com/rfanazhari/amer-prototype/domain/agentconfig"
	aggregate "github.com/rfanazhari/amer-prototype/domain/aggregate/agentconfig"
	"github.com/rfanazhari/amer-prototype/domain/valueobject"

	"github.com/rfanazhari/amer-prototype/application/repository/agentconfigport"
)

// FromCatalog copies developer-owned template identities into detached application views.
func FromCatalog(templates []catalog.Template) []AgentTemplate {
	out := make([]AgentTemplate, len(templates))
	for i, template := range templates {
		out[i] = AgentTemplate{
			TemplateID: template.ID, TemplateVersion: template.Version, BehaviorVersion: template.Behavior,
			DisplayName: template.DisplayName, WarningPolicy: template.WarningPolicy,
		}
	}
	return out
}

// FromStoredInstance copies a committed instance into a confidentiality-filtered detached view.
func FromStoredInstance(stored agentconfigport.StoredInstance) (*AgentInstance, error) {
	switch {
	case stored.HR != nil:
		snap := stored.HR.Snapshot()
		view := FromHRConfiguration(snap.Configuration)
		return instanceView(stored.Name, snap, AgentConfigurationView{HR: &view}), nil
	case stored.Document != nil:
		snap := stored.Document.Snapshot()
		view := FromDocumentConfiguration(snap.Configuration)
		return instanceView(stored.Name, snap, AgentConfigurationView{Document: &view}), nil
	default:
		return nil, aggregate.ErrInvalid
	}
}

// FromStoredRevision copies a committed revision into a confidentiality-filtered detached view.
func FromStoredRevision(stored agentconfigport.StoredRevision) (*AgentRevision, error) {
	switch {
	case stored.HR != nil:
		snap := stored.HR.Snapshot()
		view := FromHRConfiguration(snap.Configuration)
		return revisionView(snap, AgentConfigurationView{HR: &view}), nil
	case stored.Document != nil:
		snap := stored.Document.Snapshot()
		view := FromDocumentConfiguration(snap.Configuration)
		return revisionView(snap, AgentConfigurationView{Document: &view}), nil
	default:
		return nil, aggregate.ErrInvalid
	}
}

// FromValidation copies an exact-version assessment without current-policy reinterpretation.
func FromValidation(validation aggregate.Validation) *AgentValidation {
	snap := validation.Snapshot()
	findings := make([]AgentFinding, len(snap.Findings))
	for i, finding := range snap.Findings {
		findings[i] = AgentFinding{Severity: finding.Severity.String(), Code: finding.Code, Path: finding.Path, Message: finding.Message}
	}
	return &AgentValidation{
		ValidationID: snap.ID, InstanceID: snap.InstanceID, DraftVersion: snap.DraftVersion.Number(),
		TemplateID: snap.Template.Template, TemplateVersion: snap.Template.Version, BehaviorVersion: snap.Template.Behavior,
		Eligible: snap.Eligible, Findings: findings, CreatedAt: snap.CreatedAt,
	}
}

// FromHRConfiguration copies HR fields and replaces salary-reference contents with a configured flag.
func FromHRConfiguration(cfg valueobject.HRConfiguration) HRConfigurationView {
	view := HRConfigurationView{JobDescriptionRef: stableRefView(cfg.JobDescriptionRef()), CompetencyTopics: append([]string(nil), cfg.CompetencyTopics()...)}
	for _, question := range cfg.MandatoryQuestions() {
		view.MandatoryQuestions = append(view.MandatoryQuestions, MandatoryQuestionView{ID: question.ID(), Stage: question.Stage().String(), Text: question.Text()})
	}
	if stages := cfg.StageSeconds(); stages != nil {
		view.StageSeconds = &StageSecondsView{Background: stages.Background(), Screening: stages.Screening(), Compensation: stages.Compensation(), Closing: stages.Closing()}
	}
	if policy := cfg.CompensationPolicy(); policy != nil {
		view.CompensationPolicy = &CompensationPolicyView{DiscussionMode: policy.DiscussionMode(), SalaryCeilingConfigured: policy.SalaryCeilingRef() != nil}
	}
	return view
}

// FromDocumentConfiguration copies required invoice fields in catalog order.
func FromDocumentConfiguration(cfg valueobject.DocumentConfiguration) DocumentConfigurationView {
	fields := cfg.RequiredFields()
	out := make([]string, len(fields))
	for i, field := range fields {
		out[i] = field.String()
	}
	return DocumentConfigurationView{RequiredFields: out}
}

// instanceView copies template-tagged instance facts into a detached DTO.
func instanceView[C aggregate.Configuration[C]](name string, snap aggregate.InstanceSnapshot[C], configuration AgentConfigurationView) *AgentInstance {
	var active *string
	if snap.Active != nil {
		id := snap.Active.ID
		active = &id
	}
	return &AgentInstance{
		ID: snap.ID, Name: name, TemplateID: snap.Template.Template, TemplateVersion: snap.Template.Version,
		BehaviorVersion: snap.Template.Behavior, DraftVersion: snap.DraftVersion.Number(), Configuration: configuration,
		ActiveRevisionID: active, CreatedAt: snap.CreatedAt, UpdatedAt: snap.UpdatedAt,
	}
}

// revisionView copies template-tagged revision facts into a detached DTO.
func revisionView[C aggregate.Configuration[C]](snap aggregate.RevisionSnapshot[C], configuration AgentConfigurationView) *AgentRevision {
	return &AgentRevision{
		RevisionID: snap.Identity.ID, InstanceID: snap.Identity.InstanceID, RevisionNumber: snap.Identity.Number,
		SourceDraftVersion: snap.Identity.DraftVersion.Number(), ValidationID: snap.ValidationID,
		TemplateID: snap.Identity.Template.Template, TemplateVersion: snap.Identity.Template.Version,
		BehaviorVersion: snap.Identity.Template.Behavior, Configuration: configuration, PublishedAt: snap.CreatedAt,
	}
}

// stableRefView copies an optional immutable reference identity.
func stableRefView(ref *valueobject.StableRef) *StableRefView {
	if ref == nil {
		return nil
	}
	return &StableRefView{ID: ref.ID(), Version: ref.Version()}
}
