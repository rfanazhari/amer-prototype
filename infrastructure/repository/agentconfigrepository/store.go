package agentconfigrepository

import (
	"context"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/rfanazhari/amer-prototype/application/repository/agentconfigport"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// Store implements configuration persistence through a runtime-owned GORM pool.
type Store struct{ DB *gorm.DB }

var (
	_ agentconfigport.Store          = Store{}
	_ agentconfigport.InstanceReader = Store{}
	_ agentconfigport.ReplayReader   = Store{}
)

// deferConstraints postpones same-instance draft and revision foreign keys until commit.
func deferConstraints(tx *gorm.DB) error {
	return tx.Exec("SET CONSTRAINTS ALL DEFERRED").Error
}

// classifyUnique maps a publication-command unique violation onto ErrDuplicateCommand.
func classifyUnique(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" && strings.Contains(pgErr.ConstraintName, "agent_publication_commands") {
		return agentconfigport.ErrDuplicateCommand
	}
	return err
}

// notFoundOrError translates GORM absence into ErrNotFound while preserving other storage failures.
func notFoundOrError(err error) error {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return agentconfigport.ErrNotFound
	}
	return err
}

// Create atomically inserts the instance and owned initial draft, with no publication.
func (s Store) Create(ctx context.Context, stored agentconfigport.StoredInstance) error {
	instance, draft, err := fromStoredInstance(stored)
	if err != nil {
		return err
	}
	return s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := deferConstraints(tx); err != nil {
			return err
		}
		if err := tx.Create(&instance).Error; err != nil {
			return err
		}
		return tx.Create(&draft).Error
	})
}

// FindInstance restores a detached instance or classifies absence and corrupt stored state.
func (s Store) FindInstance(ctx context.Context, id string) (agentconfigport.StoredInstance, error) {
	return loadInstance(ctx, s.DB, id, false)
}

// FindRevision restores one owned immutable revision without loading history.
func (s Store) FindRevision(ctx context.Context, instanceID, revisionID string) (agentconfigport.StoredRevision, error) {
	var model RevisionModel
	err := s.DB.WithContext(ctx).Where("instance_id = ? AND id = ?", instanceID, revisionID).Take(&model).Error
	if err != nil {
		return agentconfigport.StoredRevision{}, notFoundOrError(err)
	}
	return toStoredRevision(model)
}

// FindCommand returns a detached command result from a fresh read after unique conflict or uncertain commit.
func (s Store) FindCommand(ctx context.Context, identity agentconfigport.CommandIdentity) (agentconfigport.CommandResult, bool, error) {
	return findCommand(ctx, s.DB, identity)
}

// WithinInstance locks the instance before draft, validation and revision rows and commits the callback atomically.
func (s Store) WithinInstance(ctx context.Context, id string, fn func(agentconfigport.InstanceTx) error) error {
	var committed bool
	err := s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := deferConstraints(tx); err != nil {
			return err
		}
		stored, err := loadInstance(ctx, tx, id, true)
		if err != nil {
			return err
		}
		locked, err := storedDraftVersion(stored)
		if err != nil {
			return err
		}
		if err := fn(&instanceTx{tx: tx, id: id, lockedVersion: locked, loaded: stored}); err != nil {
			return classifyUnique(err)
		}
		committed = true
		return nil
	})
	if err != nil && committed {
		return &agentconfigport.CommitUncertainError{Cause: err}
	}
	return classifyUnique(err)
}

// loadInstance reads the instance, current draft and optional active revision, optionally taking a row lock.
func loadInstance(ctx context.Context, db *gorm.DB, id string, lock bool) (agentconfigport.StoredInstance, error) {
	query := db.WithContext(ctx)
	if lock {
		query = query.Clauses(clause.Locking{Strength: "UPDATE"})
	}
	var instance InstanceModel
	if err := query.Where("id = ?", id).Take(&instance).Error; err != nil {
		return agentconfigport.StoredInstance{}, notFoundOrError(err)
	}
	var draft DraftModel
	if err := db.WithContext(ctx).Where("instance_id = ? AND version = ?", id, instance.CurrentDraftVersion).Take(&draft).Error; err != nil {
		return agentconfigport.StoredInstance{}, notFoundOrError(err)
	}
	var active *RevisionModel
	if instance.ActiveRevisionID != nil {
		var revision RevisionModel
		if err := db.WithContext(ctx).Where("instance_id = ? AND id = ?", id, *instance.ActiveRevisionID).Take(&revision).Error; err != nil {
			return agentconfigport.StoredInstance{}, technical(notFoundOrError(err))
		}
		active = &revision
	}
	return toStoredInstance(instance, draft, active)
}

// storedDraftVersion returns the persisted current draft sequence before caller mutation.
func storedDraftVersion(stored agentconfigport.StoredInstance) (int64, error) {
	switch {
	case stored.HR != nil:
		return int64(stored.HR.Snapshot().DraftVersion.Number()), nil
	case stored.Document != nil:
		return int64(stored.Document.Snapshot().DraftVersion.Number()), nil
	default:
		return 0, agentconfigport.ErrNotFound
	}
}

// findCommand joins the unique command identity to its original owning instance.
func findCommand(ctx context.Context, db *gorm.DB, identity agentconfigport.CommandIdentity) (agentconfigport.CommandResult, bool, error) {
	var row struct {
		Scope               string
		CommandKind         string `gorm:"column:command_kind"`
		RequestKey          string `gorm:"column:request_key"`
		IntentFingerprint   string `gorm:"column:intent_fingerprint"`
		ResultingRevisionID string `gorm:"column:resulting_revision_id"`
		InstanceID          string `gorm:"column:instance_id"`
	}
	err := db.WithContext(ctx).Table("agent_publication_commands").
		Select("agent_publication_commands.scope, agent_publication_commands.command_kind, agent_publication_commands.request_key, agent_publication_commands.intent_fingerprint, agent_publication_commands.resulting_revision_id, agent_revisions.instance_id").
		Joins("JOIN agent_revisions ON agent_revisions.id = agent_publication_commands.resulting_revision_id").
		Where("agent_publication_commands.scope = ? AND agent_publication_commands.command_kind = ? AND agent_publication_commands.request_key = ?", identity.Scope, identity.Kind, identity.Key).
		Take(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return agentconfigport.CommandResult{}, false, nil
	}
	if err != nil {
		return agentconfigport.CommandResult{}, false, err
	}
	return agentconfigport.CommandResult{Fingerprint: row.IntentFingerprint, InstanceID: row.InstanceID, RevisionID: row.ResultingRevisionID}, true, nil
}
