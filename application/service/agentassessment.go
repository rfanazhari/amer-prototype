package service

import (
	"context"

	catalog "github.com/rfanazhari/amer-prototype/domain/agentconfig"
	aggregate "github.com/rfanazhari/amer-prototype/domain/aggregate/agentconfig"
	"github.com/rfanazhari/amer-prototype/domain/valueobject"
)

// DraftAssessment resolves exact reference versions then runs deterministic catalog validators.
type DraftAssessment struct {
	References AgentReferenceReader[AgentReferenceFact]
}

// DraftAssessmentResult is a detached eligibility decision for one template interpretation.
type DraftAssessmentResult struct {
	TemplateID, TemplateVersion, BehaviorVersion string
	Eligible                                     bool
	Findings                                     []aggregate.Finding
}

// Catalog returns both supported versioned templates without provider calls.
func (DraftAssessment) Catalog() []catalog.Template { return catalog.Templates() }

// AssessHR validates an HR draft against locally resolved exact reference versions.
func (a DraftAssessment) AssessHR(ctx context.Context, template aggregate.TemplateRef, cfg valueobject.HRConfiguration) (DraftAssessmentResult, error) {
	desc, err := requireTemplate(template)
	if err != nil {
		return DraftAssessmentResult{}, err
	}
	refs, err := a.resolveHR(ctx, cfg)
	if err != nil {
		return DraftAssessmentResult{}, err
	}
	findings := catalog.AssessHR(cfg, refs)
	return resultFor(desc, findings), nil
}

// AssessDocument validates a Document Analyzer draft without reference or provider calls.
func (DraftAssessment) AssessDocument(_ context.Context, template aggregate.TemplateRef, cfg valueobject.DocumentConfiguration) (DraftAssessmentResult, error) {
	desc, err := requireTemplate(template)
	if err != nil {
		return DraftAssessmentResult{}, err
	}
	return resultFor(desc, catalog.AssessDocument(cfg)), nil
}

// requireTemplate rejects identities that are not in the current catalog interpretation.
func requireTemplate(template aggregate.TemplateRef) (catalog.Template, error) {
	desc, ok := catalog.Lookup(template.Template, template.Version)
	if !ok || desc.Behavior != template.Behavior {
		return catalog.Template{}, catalog.ErrUnknownTemplate
	}
	return desc, nil
}

// resultFor copies catalog identity onto a detached eligibility result.
func resultFor(desc catalog.Template, findings []aggregate.Finding) DraftAssessmentResult {
	return DraftAssessmentResult{
		TemplateID: desc.ID, TemplateVersion: desc.Version, BehaviorVersion: desc.Behavior,
		Eligible: catalog.Eligible(findings), Findings: findings,
	}
}

// resolveHR loads exact job-description and compensation versions outside the domain.
func (a DraftAssessment) resolveHR(ctx context.Context, cfg valueobject.HRConfiguration) (catalog.HRReferenceContext, error) {
	var refs catalog.HRReferenceContext
	jd, err := resolveOptional(ctx, a.References, ReferenceKindJobDescription, cfg.JobDescriptionRef())
	if err != nil {
		return catalog.HRReferenceContext{}, err
	}
	refs.JobDescription = jd
	if cfg.CompensationPolicy() == nil {
		return refs, nil
	}
	ceiling, err := resolveOptional(ctx, a.References, ReferenceKindCompensationPolicy, cfg.CompensationPolicy().SalaryCeilingRef())
	if err != nil {
		return catalog.HRReferenceContext{}, err
	}
	refs.SalaryCeiling = ceiling
	return refs, nil
}

// resolveOptional returns missing readiness when no reference is stored and preserves exact requested versions.
func resolveOptional(ctx context.Context, reader AgentReferenceReader[AgentReferenceFact], kind string, ref *valueobject.StableRef) (catalog.ReferenceReadiness, error) {
	if ref == nil {
		return catalog.ReferenceReadiness{}, nil
	}
	fact, found, err := reader.Resolve(ctx, AgentReference{Kind: kind, ID: ref.ID(), Version: ref.Version()})
	if err != nil {
		return catalog.ReferenceReadiness{}, err
	}
	if !found {
		return catalog.ReferenceReadiness{}, nil
	}
	if fact.ID != ref.ID() || fact.Version != ref.Version() {
		return catalog.ReferenceReadiness{}, ErrReferenceIdentity
	}
	return catalog.ReferenceReadiness{Found: true, Ready: fact.Ready, ID: fact.ID, Version: fact.Version, Role: fact.Role}, nil
}
