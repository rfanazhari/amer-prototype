package agentconfigusecase

import (
	"context"

	"github.com/rfanazhari/amer-prototype/application/dto"
	"github.com/rfanazhari/amer-prototype/application/repository/agentconfigport"
	"github.com/rfanazhari/amer-prototype/application/service"
	aggregate "github.com/rfanazhari/amer-prototype/domain/aggregate/agentconfig"
)

// Replace performs an exact-version full draft replacement.
type Replace struct {
	Authority service.AgentConfigurationAuthority
	Instances agentconfigport.Store
	Clock     service.Clock
}

// Execute authorizes replacement and returns a detached instance only after commit.
func (u Replace) Execute(ctx context.Context, in ReplaceInput) (*dto.AgentInstance, error) {
	if _, err := authorize(ctx, u.Authority, service.AgentConfigurationReplace, in.InstanceID); err != nil {
		return nil, err
	}
	expected, err := aggregate.NewDraftVersion(in.ExpectedDraftVersion)
	if err != nil {
		return nil, service.ErrDraftConflict
	}
	now := trustedNow(u.Clock)
	var stored agentconfigport.StoredInstance
	err = u.Instances.WithinInstance(ctx, in.InstanceID, func(tx agentconfigport.InstanceTx) error {
		current, err := tx.Instance(ctx)
		if err != nil {
			return err
		}
		if err := replaceStored(&current, in, expected, now); err != nil {
			return err
		}
		if err := tx.SaveInstance(ctx, current); err != nil {
			return err
		}
		stored = current
		return nil
	})
	if err != nil {
		return nil, err
	}
	return dto.FromStoredInstance(stored)
}
