package agentconfigrepository

import (
	"context"
	"time"

	"github.com/rfanazhari/amer-prototype/application/repository/agentconfigport"
	aggregate "github.com/rfanazhari/amer-prototype/domain/aggregate/agentconfig"
	"gorm.io/gorm"
)

// instanceTx is the instance-owned transaction surface used after the instance row lock.
type instanceTx struct {
	tx            *gorm.DB
	id            string
	lockedVersion int64 // draft sequence captured at lock time, before shared aggregate mutation
	loaded        agentconfigport.StoredInstance
}

var _ agentconfigport.InstanceTx = (*instanceTx)(nil)

// Instance returns the locked instance and current draft as a detached domain value.
func (t *instanceTx) Instance(context.Context) (agentconfigport.StoredInstance, error) {
	return t.loaded, nil
}

// SaveInstance persists explicit zero, empty, false and null fields together with the aggregate-selected pointer.
func (t *instanceTx) SaveInstance(ctx context.Context, stored agentconfigport.StoredInstance) error {
	instance, draft, err := fromStoredInstance(stored)
	if err != nil {
		return err
	}
	if instance.ID != t.id {
		return agentconfigport.ErrNotFound
	}
	if draft.Version == t.lockedVersion {
		if err := t.tx.WithContext(ctx).Exec(
			`UPDATE agent_drafts SET configuration = ?, updated_at = ? WHERE instance_id = ? AND version = ?`,
			[]byte(draft.Configuration), draft.UpdatedAt, t.id, t.lockedVersion,
		).Error; err != nil {
			return err
		}
	} else if err := t.tx.WithContext(ctx).Exec(
		`INSERT INTO agent_drafts (instance_id, version, configuration, created_at, updated_at) VALUES (?, ?, ?, ?, ?)`,
		t.id, draft.Version, []byte(draft.Configuration), draft.CreatedAt, draft.UpdatedAt,
	).Error; err != nil {
		return err
	}
	if err := t.tx.WithContext(ctx).Exec(
		`UPDATE agent_instances SET name = ?, template_id = ?, template_version = ?, behavior_version = ?, current_draft_version = ?, active_revision_id = ?, updated_at = ? WHERE id = ?`,
		instance.Name, instance.TemplateID, instance.TemplateVersion, instance.BehaviorVersion, instance.CurrentDraftVersion, instance.ActiveRevisionID, instance.UpdatedAt, t.id,
	).Error; err != nil {
		return err
	}
	t.lockedVersion = draft.Version
	t.loaded = stored
	return nil
}

// FindValidation loads one immutable assessment belonging to the locked instance.
func (t *instanceTx) FindValidation(ctx context.Context, id string) (aggregate.Validation, error) {
	var model ValidationModel
	if err := t.tx.WithContext(ctx).Where("id = ?", id).Take(&model).Error; err != nil {
		return aggregate.Validation{}, notFoundOrError(err)
	}
	if model.InstanceID != t.id {
		return aggregate.Validation{}, agentconfigport.ErrNotFound
	}
	return toValidation(model)
}

// InsertValidation records an exact-version assessment after the domain version check.
func (t *instanceTx) InsertValidation(ctx context.Context, validation aggregate.Validation) error {
	model, err := fromValidation(validation)
	if err != nil {
		return err
	}
	if model.InstanceID != t.id {
		return agentconfigport.ErrNotFound
	}
	return t.tx.WithContext(ctx).Create(&model).Error
}

// FindCommand resolves replay before draft and version checks without changing active selection.
func (t *instanceTx) FindCommand(ctx context.Context, identity agentconfigport.CommandIdentity) (agentconfigport.CommandResult, bool, error) {
	return findCommand(ctx, t.tx, identity)
}

// InsertRevision inserts an immutable owned revision; revision updates are not exposed.
func (t *instanceTx) InsertRevision(ctx context.Context, stored agentconfigport.StoredRevision) error {
	if revisionInstanceID(stored) != t.id {
		return agentconfigport.ErrNotFound
	}
	model, err := fromStoredRevision(stored)
	if err != nil {
		return err
	}
	return t.tx.WithContext(ctx).Create(&model).Error
}

// InsertCommand stores the unique identity and original result in this transaction.
func (t *instanceTx) InsertCommand(ctx context.Context, identity agentconfigport.CommandIdentity, result agentconfigport.CommandResult) error {
	if result.InstanceID != t.id {
		return agentconfigport.ErrNotFound
	}
	model := CommandModel{
		Scope: identity.Scope, CommandKind: identity.Kind, RequestKey: identity.Key,
		IntentFingerprint: result.Fingerprint, ResultingRevisionID: result.RevisionID, CreatedAt: time.Now().UTC(),
	}
	if err := t.tx.WithContext(ctx).Create(&model).Error; err != nil {
		return classifyUnique(err)
	}
	return nil
}
