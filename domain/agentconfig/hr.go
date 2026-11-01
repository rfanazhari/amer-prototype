package agentconfig

import (
	"github.com/rfanazhari/amer-prototype/domain/aggregate/agentconfig"
	"github.com/rfanazhari/amer-prototype/domain/valueobject"
)

const (
	questionCapacitySeconds = 30
	followupSpareSeconds    = 30
	minPublicationTotal     = 300
	maxPublicationTotal     = 600
	demoJobRole             = "Backend Engineer"
)

// ReferenceReadiness is a detached exact-version reference fact, not a live provider check.
type ReferenceReadiness struct {
	Found, Ready      bool
	ID, Version, Role string
}

// HRReferenceContext holds locally resolved job-description and compensation facts.
type HRReferenceContext struct {
	JobDescription, SalaryCeiling ReferenceReadiness
}

// AssessHR records deterministic HR findings without model or provider calls.
func AssessHR(cfg valueobject.HRConfiguration, refs HRReferenceContext) []agentconfig.Finding {
	return nil
}
