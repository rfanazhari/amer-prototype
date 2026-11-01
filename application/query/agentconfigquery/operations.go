package agentconfigquery

import (
	"context"

	"github.com/rfanazhari/amer-prototype/application/dto"
)

// CatalogOperation is the template catalog query contract consumed by delivery.
type CatalogOperation interface {
	// Execute authorizes collection read and returns detached catalog entries.
	Execute(context.Context) ([]dto.AgentTemplate, error)
}

// InstanceOperation is the instance-read query contract consumed by delivery.
type InstanceOperation interface {
	// Execute authorizes read before disclosing a stored instance.
	Execute(context.Context, string) (*dto.AgentInstance, error)
}

// RevisionOperation is the revision-read query contract consumed by delivery.
type RevisionOperation interface {
	// Execute authorizes read before disclosing a stored revision.
	Execute(context.Context, string, string) (*dto.AgentRevision, error)
}
