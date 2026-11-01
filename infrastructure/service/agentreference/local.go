package agentreference

import (
	"context"

	"github.com/rfanazhari/amer-prototype/application/service"
)

const (
	demoJobDescriptionID = "11111111-1111-4111-8111-111111111111"
	demoSalaryCeilingID  = "22222222-2222-4222-8222-222222222222"
	demoReferenceVersion = "1"
)

// Local resolves the approved synthetic job-description and compensation versions without provider I/O.
type Local struct{}

// Resolve returns detached facts for the demo Backend Engineer JD and confidential ceiling, or a missing result.
func (Local) Resolve(_ context.Context, ref service.AgentReference) (service.AgentReferenceFact, bool, error) {
	switch {
	case ref.Kind == service.ReferenceKindJobDescription && ref.ID == demoJobDescriptionID && ref.Version == demoReferenceVersion:
		return service.AgentReferenceFact{Kind: ref.Kind, ID: ref.ID, Version: ref.Version, Role: service.DemoJobRole, Ready: true}, true, nil
	case ref.Kind == service.ReferenceKindCompensationPolicy && ref.ID == demoSalaryCeilingID && ref.Version == demoReferenceVersion:
		return service.AgentReferenceFact{Kind: ref.Kind, ID: ref.ID, Version: ref.Version, Ready: true}, true, nil
	default:
		return service.AgentReferenceFact{}, false, nil
	}
}
