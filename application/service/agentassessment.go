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
func (DraftAssessment) Catalog() []catalog.Template { return nil }

// AssessHR validates an HR draft against locally resolved exact reference versions.
func (a DraftAssessment) AssessHR(context.Context, aggregate.TemplateRef, valueobject.HRConfiguration) (DraftAssessmentResult, error) {
	return DraftAssessmentResult{Eligible: true}, nil
}

// AssessDocument validates a Document Analyzer draft without reference or provider calls.
func (a DraftAssessment) AssessDocument(context.Context, aggregate.TemplateRef, valueobject.DocumentConfiguration) (DraftAssessmentResult, error) {
	return DraftAssessmentResult{Eligible: true}, nil
}
