package agentconfigusecase

import (
	"context"

	"github.com/google/uuid"
	"github.com/rfanazhari/amer-prototype/application/dto"
	"github.com/rfanazhari/amer-prototype/application/repository/agentconfigport"
	"github.com/rfanazhari/amer-prototype/application/service"
	catalog "github.com/rfanazhari/amer-prototype/domain/agentconfig"
)

// Create creates an unpublished instance for a catalog template version.
type Create struct {
	Authority service.AgentConfigurationAuthority
	Instances agentconfigport.Store
	Clock     service.Clock
}

// Execute authorizes collection create and returns a detached instance only after commit.
func (u Create) Execute(ctx context.Context, in CreateInput) (*dto.AgentInstance, error) {
	if _, err := authorize(ctx, u.Authority, service.AgentConfigurationCreate, ""); err != nil {
		return nil, err
	}
	name, err := service.CanonicalAdministrativeName(in.Name)
	if err != nil {
		return nil, err
	}
	template, ok := catalog.Lookup(in.TemplateID, in.TemplateVersion)
	if !ok {
		return nil, catalog.ErrUnknownTemplate
	}
	stored, err := newStoredInstance(uuid.NewString(), name, template, in, trustedNow(u.Clock))
	if err != nil {
		return nil, err
	}
	if err := u.Instances.Create(ctx, stored); err != nil {
		return nil, err
	}
	return dto.FromStoredInstance(stored)
}
