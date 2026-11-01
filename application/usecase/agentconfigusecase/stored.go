package agentconfigusecase

import (
	"context"
	"errors"
	"time"

	"github.com/rfanazhari/amer-prototype/application/repository/agentconfigport"
	"github.com/rfanazhari/amer-prototype/application/service"
	catalog "github.com/rfanazhari/amer-prototype/domain/agentconfig"
	aggregate "github.com/rfanazhari/amer-prototype/domain/aggregate/agentconfig"
	"github.com/rfanazhari/amer-prototype/domain/valueobject"
)

// newStoredInstance constructs an unpublished draft for the selected catalog template.
func newStoredInstance(id, name string, template catalog.Template, in CreateInput, now time.Time) (agentconfigport.StoredInstance, error) {
	version, err := aggregate.NewDraftVersion(1)
	if err != nil {
		return agentconfigport.StoredInstance{}, err
	}
	switch template.ID {
	case catalog.TemplateHRInterviewer:
		input := valueobject.HRInput{}
		if in.HR != nil {
			input = *in.HR
		}
		cfg, err := valueobject.NewHRConfiguration(input)
		if err != nil {
			return agentconfigport.StoredInstance{}, err
		}
		instance, err := aggregate.NewInstance(aggregate.InstanceSnapshot[valueobject.HRConfiguration]{
			ID: id, Template: template.Ref(), DraftVersion: version, Configuration: cfg, CreatedAt: now, UpdatedAt: now,
		})
		if err != nil {
			return agentconfigport.StoredInstance{}, err
		}
		return agentconfigport.StoredInstance{Name: name, HR: instance}, nil
	case catalog.TemplateDocumentAnalyzer:
		cfg, err := valueobject.NewDocumentConfiguration(in.DocumentFields)
		if err != nil {
			return agentconfigport.StoredInstance{}, err
		}
		instance, err := aggregate.NewInstance(aggregate.InstanceSnapshot[valueobject.DocumentConfiguration]{
			ID: id, Template: template.Ref(), DraftVersion: version, Configuration: cfg, CreatedAt: now, UpdatedAt: now,
		})
		if err != nil {
			return agentconfigport.StoredInstance{}, err
		}
		return agentconfigport.StoredInstance{Name: name, Document: instance}, nil
	default:
		return agentconfigport.StoredInstance{}, catalog.ErrUnknownTemplate
	}
}

// replaceStored applies an exact-version full replacement on the locked template-tagged instance.
func replaceStored(stored *agentconfigport.StoredInstance, in ReplaceInput, expected aggregate.DraftVersion, now time.Time) error {
	if stored.HR != nil {
		input := valueobject.HRInput{}
		if in.HR != nil {
			input = *in.HR
		}
		cfg, err := valueobject.NewHRConfiguration(input)
		if err != nil {
			return err
		}
		return mapDraftError(stored.HR.ReplaceDraft(expected, cfg, now))
	}
	if stored.Document != nil {
		cfg, err := valueobject.NewDocumentConfiguration(in.DocumentFields)
		if err != nil {
			return err
		}
		return mapDraftError(stored.Document.ReplaceDraft(expected, cfg, now))
	}
	return aggregate.ErrInvalid
}

// assessStored resolves references outside the transaction and runs the matching catalog validator.
func assessStored(ctx context.Context, assessment service.DraftAssessment, stored agentconfigport.StoredInstance) (service.DraftAssessmentResult, error) {
	switch {
	case stored.HR != nil:
		snap := stored.HR.Snapshot()
		return assessment.AssessHR(ctx, snap.Template, snap.Configuration)
	case stored.Document != nil:
		snap := stored.Document.Snapshot()
		return assessment.AssessDocument(ctx, snap.Template, snap.Configuration)
	default:
		return service.DraftAssessmentResult{}, aggregate.ErrInvalid
	}
}

// currentDraftNumber returns the locked instance draft version.
func currentDraftNumber(stored agentconfigport.StoredInstance) (uint64, error) {
	switch {
	case stored.HR != nil:
		return stored.HR.Snapshot().DraftVersion.Number(), nil
	case stored.Document != nil:
		return stored.Document.Snapshot().DraftVersion.Number(), nil
	default:
		return 0, aggregate.ErrInvalid
	}
}

// lockedTemplate returns the immutable template reference of the locked instance.
func lockedTemplate(stored agentconfigport.StoredInstance) (aggregate.TemplateRef, error) {
	switch {
	case stored.HR != nil:
		return stored.HR.Snapshot().Template, nil
	case stored.Document != nil:
		return stored.Document.Snapshot().Template, nil
	default:
		return aggregate.TemplateRef{}, aggregate.ErrInvalid
	}
}

// recordValidation constructs an immutable assessment for the locked expected draft.
func recordValidation(id, instanceID string, draft uint64, template aggregate.TemplateRef, result service.DraftAssessmentResult, now time.Time) (aggregate.Validation, error) {
	version, err := aggregate.NewDraftVersion(draft)
	if err != nil {
		return aggregate.Validation{}, err
	}
	return aggregate.NewValidation(aggregate.ValidationSnapshot{
		ID: id, InstanceID: instanceID, DraftVersion: version, Template: template,
		Eligible: result.Eligible, Findings: append([]aggregate.Finding(nil), result.Findings...), CreatedAt: now,
	})
}

// publishStored freezes the locked eligible draft after replay has already been ruled out.
func publishStored(stored *agentconfigport.StoredInstance, revisionID string, validation aggregate.Validation, now time.Time) (agentconfigport.StoredRevision, error) {
	if !validation.Snapshot().Eligible {
		return agentconfigport.StoredRevision{}, service.ErrPublicationBlocked
	}
	switch {
	case stored.HR != nil:
		revision, err := stored.HR.Publish(revisionID, validation, now)
		if err != nil {
			return agentconfigport.StoredRevision{}, mapPublishError(err)
		}
		return agentconfigport.StoredRevision{HR: &revision}, nil
	case stored.Document != nil:
		revision, err := stored.Document.Publish(revisionID, validation, now)
		if err != nil {
			return agentconfigport.StoredRevision{}, mapPublishError(err)
		}
		return agentconfigport.StoredRevision{Document: &revision}, nil
	default:
		return agentconfigport.StoredRevision{}, aggregate.ErrInvalid
	}
}

// mapDraftError classifies stale replacement as a draft conflict.
func mapDraftError(err error) error {
	if errors.Is(err, aggregate.ErrStale) {
		return service.ErrDraftConflict
	}
	return err
}

// mapPublishError classifies validation mismatches without disclosing foreign owners.
func mapPublishError(err error) error {
	if errors.Is(err, aggregate.ErrStale) {
		return service.ErrDraftConflict
	}
	if errors.Is(err, aggregate.ErrValidation) {
		return service.ErrValidationConflict
	}
	return err
}
