package agentconfigusecase

import (
	"context"

	"github.com/google/uuid"
	"github.com/rfanazhari/amer-prototype/application/dto"
	"github.com/rfanazhari/amer-prototype/application/repository/agentconfigport"
	"github.com/rfanazhari/amer-prototype/application/service"
	aggregate "github.com/rfanazhari/amer-prototype/domain/aggregate/agentconfig"
)

// Validate records a deterministic assessment against an exact current draft version.
type Validate struct {
	Authority  service.AgentConfigurationAuthority
	Reader     agentconfigport.InstanceReader
	Instances  agentconfigport.Store
	Assessment service.DraftAssessment
	Clock      service.Clock
}

// Execute authorizes validation and records findings only for the locked expected draft.
func (u Validate) Execute(ctx context.Context, in ValidateInput) (*dto.AgentValidation, error) {
	if _, err := authorize(ctx, u.Authority, service.AgentConfigurationValidate, in.InstanceID); err != nil {
		return nil, err
	}
	current, err := u.Reader.FindInstance(ctx, in.InstanceID)
	if err != nil {
		return nil, err
	}
	result, err := assessStored(ctx, u.Assessment, current)
	if err != nil {
		return nil, err
	}
	var recorded aggregate.Validation
	err = u.Instances.WithinInstance(ctx, in.InstanceID, func(tx agentconfigport.InstanceTx) error {
		locked, err := tx.Instance(ctx)
		if err != nil {
			return err
		}
		draft, err := currentDraftNumber(locked)
		if err != nil {
			return err
		}
		if draft != in.ExpectedDraftVersion {
			return service.ErrDraftConflict
		}
		template, err := lockedTemplate(locked)
		if err != nil {
			return err
		}
		validation, err := recordValidation(uuid.NewString(), in.InstanceID, draft, template, result, trustedNow(u.Clock))
		if err != nil {
			return err
		}
		if err := tx.InsertValidation(ctx, validation); err != nil {
			return err
		}
		recorded = validation
		return nil
	})
	if err != nil {
		return nil, err
	}
	return dto.FromValidation(recorded), nil
}
