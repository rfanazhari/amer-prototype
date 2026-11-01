package agentconfigrepository

import (
	"time"

	"github.com/rfanazhari/amer-prototype/application/repository/agentconfigport"
	catalog "github.com/rfanazhari/amer-prototype/domain/agentconfig"
	aggregate "github.com/rfanazhari/amer-prototype/domain/aggregate/agentconfig"
	"github.com/rfanazhari/amer-prototype/domain/valueobject"
	"github.com/rfanazhari/amer-prototype/pkg/errs"
)

// utcTime copies a stored timestamp into canonical UTC.
func utcTime(value time.Time) time.Time { return value.UTC() }

// copyString returns an independently owned optional identity.
func copyString(value *string) *string {
	if value == nil {
		return nil
	}
	copied := *value
	return &copied
}

// technical wraps restoration failures so delivery can classify them before nested domain errors.
func technical(err error) error {
	if err == nil {
		return nil
	}
	return errs.Technical{Cause: err}
}

// ensureRestorationSupported rejects stored identities outside the deployed restoration catalog.
func ensureRestorationSupported(ref aggregate.TemplateRef) error {
	if !catalog.RestorationSupported(ref) {
		return technical(catalog.ErrUnsupportedStoredTemplate)
	}
	return nil
}

// fromStoredInstance maps a detached instance onto instance and draft rows.
func fromStoredInstance(stored agentconfigport.StoredInstance) (InstanceModel, DraftModel, error) {
	switch {
	case stored.HR != nil && stored.Document == nil:
		snap := stored.HR.Snapshot()
		body, err := encodeHRConfiguration(snap.Configuration)
		if err != nil {
			return InstanceModel{}, DraftModel{}, technical(err)
		}
		return instanceRow(stored.Name, snap.ID, snap.Template, snap.DraftVersion.Number(), snap.Active, snap.CreatedAt, snap.UpdatedAt), draftRow(snap.ID, snap.DraftVersion.Number(), body, snap.CreatedAt, snap.UpdatedAt), nil
	case stored.Document != nil && stored.HR == nil:
		snap := stored.Document.Snapshot()
		body, err := encodeDocumentConfiguration(snap.Configuration)
		if err != nil {
			return InstanceModel{}, DraftModel{}, technical(err)
		}
		return instanceRow(stored.Name, snap.ID, snap.Template, snap.DraftVersion.Number(), snap.Active, snap.CreatedAt, snap.UpdatedAt), draftRow(snap.ID, snap.DraftVersion.Number(), body, snap.CreatedAt, snap.UpdatedAt), nil
	default:
		return InstanceModel{}, DraftModel{}, technical(aggregate.ErrInvalid)
	}
}

// instanceRow copies instance facts onto the storage model.
func instanceRow(name, id string, template aggregate.TemplateRef, draft uint64, active *aggregate.RevisionIdentity, created, updated time.Time) InstanceModel {
	row := InstanceModel{
		ID: id, Name: name, TemplateID: template.Template, TemplateVersion: template.Version, BehaviorVersion: template.Behavior,
		CurrentDraftVersion: int64(draft), CreatedAt: utcTime(created), UpdatedAt: utcTime(updated),
	}
	if active != nil {
		identity := active.ID
		row.ActiveRevisionID = copyString(&identity)
	}
	return row
}

// draftRow copies the current draft onto the storage model.
func draftRow(id string, version uint64, body JSONDocument, created, updated time.Time) DraftModel {
	return DraftModel{InstanceID: id, Version: int64(version), Configuration: body, CreatedAt: utcTime(created), UpdatedAt: utcTime(updated)}
}

// toStoredInstance restores a template-tagged instance from instance, draft and optional active revision rows.
func toStoredInstance(instance InstanceModel, draft DraftModel, active *RevisionModel) (agentconfigport.StoredInstance, error) {
	template := aggregate.TemplateRef{Template: instance.TemplateID, Version: instance.TemplateVersion, Behavior: instance.BehaviorVersion}
	if err := ensureRestorationSupported(template); err != nil {
		return agentconfigport.StoredInstance{}, err
	}
	version, err := aggregate.NewDraftVersion(uint64(instance.CurrentDraftVersion))
	if err != nil {
		return agentconfigport.StoredInstance{}, technical(err)
	}
	if draft.InstanceID != instance.ID || draft.Version != instance.CurrentDraftVersion {
		return agentconfigport.StoredInstance{}, technical(aggregate.ErrInvalid)
	}
	identity, err := activeIdentity(instance, active)
	if err != nil {
		return agentconfigport.StoredInstance{}, err
	}
	switch instance.TemplateID {
	case catalog.TemplateHRInterviewer:
		cfg, err := decodeHRConfiguration(draft.Configuration)
		if err != nil {
			return agentconfigport.StoredInstance{}, technical(err)
		}
		restored, err := aggregate.RestoreInstance(aggregate.InstanceSnapshot[valueobject.HRConfiguration]{
			ID: instance.ID, Template: template, DraftVersion: version, Configuration: cfg, Active: identity,
			CreatedAt: utcTime(instance.CreatedAt), UpdatedAt: utcTime(instance.UpdatedAt),
		})
		if err != nil {
			return agentconfigport.StoredInstance{}, technical(err)
		}
		return agentconfigport.StoredInstance{Name: instance.Name, HR: restored}, nil
	case catalog.TemplateDocumentAnalyzer:
		cfg, err := decodeDocumentConfiguration(draft.Configuration)
		if err != nil {
			return agentconfigport.StoredInstance{}, technical(err)
		}
		restored, err := aggregate.RestoreInstance(aggregate.InstanceSnapshot[valueobject.DocumentConfiguration]{
			ID: instance.ID, Template: template, DraftVersion: version, Configuration: cfg, Active: identity,
			CreatedAt: utcTime(instance.CreatedAt), UpdatedAt: utcTime(instance.UpdatedAt),
		})
		if err != nil {
			return agentconfigport.StoredInstance{}, technical(err)
		}
		return agentconfigport.StoredInstance{Name: instance.Name, Document: restored}, nil
	default:
		return agentconfigport.StoredInstance{}, technical(aggregate.ErrInvalid)
	}
}

// activeIdentity restores the owned selection or reports corrupt stored pointers.
func activeIdentity(instance InstanceModel, active *RevisionModel) (*aggregate.RevisionIdentity, error) {
	if instance.ActiveRevisionID == nil {
		return nil, nil
	}
	if active == nil || active.ID != *instance.ActiveRevisionID || active.InstanceID != instance.ID {
		return nil, technical(aggregate.ErrInvalid)
	}
	draftVersion, err := aggregate.NewDraftVersion(uint64(active.SourceDraftVersion))
	if err != nil {
		return nil, technical(err)
	}
	return &aggregate.RevisionIdentity{
		ID: active.ID, InstanceID: active.InstanceID, Number: uint64(active.RevisionNumber), DraftVersion: draftVersion,
		Template: aggregate.TemplateRef{Template: active.TemplateID, Version: active.TemplateVersion, Behavior: active.BehaviorVersion},
	}, nil
}

// fromStoredRevision maps an immutable revision onto a storage row.
func fromStoredRevision(stored agentconfigport.StoredRevision) (RevisionModel, error) {
	switch {
	case stored.HR != nil && stored.Document == nil:
		snap := stored.HR.Snapshot()
		body, err := encodeHRConfiguration(snap.Configuration)
		if err != nil {
			return RevisionModel{}, technical(err)
		}
		return revisionRow(snap.Identity, snap.ValidationID, body, snap.CreatedAt), nil
	case stored.Document != nil && stored.HR == nil:
		snap := stored.Document.Snapshot()
		body, err := encodeDocumentConfiguration(snap.Configuration)
		if err != nil {
			return RevisionModel{}, technical(err)
		}
		return revisionRow(snap.Identity, snap.ValidationID, body, snap.CreatedAt), nil
	default:
		return RevisionModel{}, technical(aggregate.ErrInvalid)
	}
}

// revisionRow copies frozen revision facts onto the storage model.
func revisionRow(identity aggregate.RevisionIdentity, validationID string, body JSONDocument, created time.Time) RevisionModel {
	return RevisionModel{
		ID: identity.ID, InstanceID: identity.InstanceID, RevisionNumber: int64(identity.Number),
		SourceDraftVersion: int64(identity.DraftVersion.Number()), ValidationID: validationID,
		TemplateID: identity.Template.Template, TemplateVersion: identity.Template.Version, BehaviorVersion: identity.Template.Behavior,
		Configuration: body, CreatedAt: utcTime(created),
	}
}

// toStoredRevision restores a template-tagged revision from one row.
func toStoredRevision(model RevisionModel) (agentconfigport.StoredRevision, error) {
	template := aggregate.TemplateRef{Template: model.TemplateID, Version: model.TemplateVersion, Behavior: model.BehaviorVersion}
	if err := ensureRestorationSupported(template); err != nil {
		return agentconfigport.StoredRevision{}, err
	}
	draft, err := aggregate.NewDraftVersion(uint64(model.SourceDraftVersion))
	if err != nil {
		return agentconfigport.StoredRevision{}, technical(err)
	}
	identity := aggregate.RevisionIdentity{
		ID: model.ID, InstanceID: model.InstanceID, Number: uint64(model.RevisionNumber), DraftVersion: draft,
		Template: aggregate.TemplateRef{Template: model.TemplateID, Version: model.TemplateVersion, Behavior: model.BehaviorVersion},
	}
	switch model.TemplateID {
	case catalog.TemplateHRInterviewer:
		cfg, err := decodeHRConfiguration(model.Configuration)
		if err != nil {
			return agentconfigport.StoredRevision{}, technical(err)
		}
		restored, err := aggregate.RestoreRevision(aggregate.RevisionSnapshot[valueobject.HRConfiguration]{
			Identity: identity, ValidationID: model.ValidationID, Configuration: cfg, CreatedAt: utcTime(model.CreatedAt),
		})
		if err != nil {
			return agentconfigport.StoredRevision{}, technical(err)
		}
		return agentconfigport.StoredRevision{HR: &restored}, nil
	case catalog.TemplateDocumentAnalyzer:
		cfg, err := decodeDocumentConfiguration(model.Configuration)
		if err != nil {
			return agentconfigport.StoredRevision{}, technical(err)
		}
		restored, err := aggregate.RestoreRevision(aggregate.RevisionSnapshot[valueobject.DocumentConfiguration]{
			Identity: identity, ValidationID: model.ValidationID, Configuration: cfg, CreatedAt: utcTime(model.CreatedAt),
		})
		if err != nil {
			return agentconfigport.StoredRevision{}, technical(err)
		}
		return agentconfigport.StoredRevision{Document: &restored}, nil
	default:
		return agentconfigport.StoredRevision{}, technical(aggregate.ErrInvalid)
	}
}

// fromValidation maps an immutable assessment onto a storage row.
func fromValidation(validation aggregate.Validation) (ValidationModel, error) {
	snap := validation.Snapshot()
	body, err := encodeFindings(snap.Findings)
	if err != nil {
		return ValidationModel{}, technical(err)
	}
	return ValidationModel{
		ID: snap.ID, InstanceID: snap.InstanceID, DraftVersion: int64(snap.DraftVersion.Number()),
		TemplateID: snap.Template.Template, TemplateVersion: snap.Template.Version, BehaviorVersion: snap.Template.Behavior,
		Eligible: snap.Eligible, Findings: body, CreatedAt: utcTime(snap.CreatedAt),
	}, nil
}

// toValidation restores an assessment from one row.
func toValidation(model ValidationModel) (aggregate.Validation, error) {
	template := aggregate.TemplateRef{Template: model.TemplateID, Version: model.TemplateVersion, Behavior: model.BehaviorVersion}
	if err := ensureRestorationSupported(template); err != nil {
		return aggregate.Validation{}, err
	}
	draft, err := aggregate.NewDraftVersion(uint64(model.DraftVersion))
	if err != nil {
		return aggregate.Validation{}, technical(err)
	}
	findings, err := decodeFindings(model.Findings)
	if err != nil {
		return aggregate.Validation{}, technical(err)
	}
	restored, err := aggregate.RestoreValidation(aggregate.ValidationSnapshot{
		ID: model.ID, InstanceID: model.InstanceID, DraftVersion: draft,
		Template: aggregate.TemplateRef{Template: model.TemplateID, Version: model.TemplateVersion, Behavior: model.BehaviorVersion},
		Eligible: model.Eligible, Findings: findings, CreatedAt: utcTime(model.CreatedAt),
	})
	if err != nil {
		return aggregate.Validation{}, technical(err)
	}
	return restored, nil
}

// revisionInstanceID returns the owning instance of a stored revision.
func revisionInstanceID(stored agentconfigport.StoredRevision) string {
	if stored.HR != nil {
		return stored.HR.Snapshot().Identity.InstanceID
	}
	if stored.Document != nil {
		return stored.Document.Snapshot().Identity.InstanceID
	}
	return ""
}
