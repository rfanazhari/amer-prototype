package executionrepository

import (
	"context"
	"database/sql"
	"errors"
	"reflect"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/rfanazhari/amer-prototype/application/repository/executionport"
	"github.com/rfanazhari/amer-prototype/domain/aggregate/execution"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// Store persists executions and the assignment lock through a shared GORM database pool.
type Store struct{ DB *gorm.DB }

var (
	_ executionport.AssignmentLock = Store{}
	_ executionport.Transactor     = Store{}
	_ executionport.ExecutionStore = Store{}
)

// txKey carries the open transaction through the context handed to an assignment callback.
type txKey struct{}

// nonterminalStates are the only states from which a terminal commit may move.
var nonterminalStates = []string{string(execution.StateCreated), string(execution.StateRunning)}

// classify maps PostgreSQL constraint failures to the port's safe error categories.
func classify(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "23505":
			return executionport.ErrAlreadyExists
		case "23503":
			return executionport.ErrDependencyUnavailable
		}
	}
	return err
}

// session returns the context transaction when present, otherwise the pool, bound to ctx.
func (s Store) session(ctx context.Context) *gorm.DB {
	if tx, ok := ctx.Value(txKey{}).(*gorm.DB); ok {
		return tx.WithContext(ctx)
	}
	return s.DB.WithContext(ctx)
}

// inTx joins the transaction carried by ctx or opens one, so callers always get all-or-nothing writes.
func (s Store) inTx(ctx context.Context, fn func(tx *gorm.DB) error) error {
	if tx, ok := ctx.Value(txKey{}).(*gorm.DB); ok {
		return fn(tx.WithContext(ctx))
	}
	return s.DB.WithContext(ctx).Transaction(fn)
}

// WithAssignment locks the assignment row before any execution lock and runs fn in the same transaction.
// The lock statement matches the one the access adapter already uses. An absent assignment is ErrNotFound.
func (s Store) WithAssignment(ctx context.Context, assignmentID string, fn func(context.Context, executionport.AssignmentRow) error) error {
	return s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec("SELECT id FROM interview_assignments WHERE id = ? FOR UPDATE", assignmentID).Error; err != nil {
			return err
		}
		var (
			row                    executionport.AssignmentRow
			revokedAt, completedAt sql.NullTime
		)
		err := tx.Raw(
			"SELECT id, instance_id, published_revision_id, access_revoked_at, completed_at FROM interview_assignments WHERE id = ?",
			assignmentID,
		).Row().Scan(&row.AssignmentID, &row.InstanceID, &row.RevisionID, &revokedAt, &completedAt)
		if errors.Is(err, sql.ErrNoRows) {
			return executionport.ErrNotFound
		}
		if err != nil {
			return err
		}
		if revokedAt.Valid {
			row.AccessRevokedAt = utcPointer(&revokedAt.Time)
		}
		if completedAt.Valid {
			row.CompletedAt = utcPointer(&completedAt.Time)
		}
		return fn(context.WithValue(ctx, txKey{}, tx), row)
	})
}

// insertExecution writes the shared execution row and then its initial transition.
func insertExecution(tx *gorm.DB, snap execution.Snapshot, fact execution.TransitionFact) error {
	model := fromSnapshot(snap)
	if err := tx.Create(&model).Error; err != nil {
		return classify(err)
	}
	transition := fromTransition(snap.ID, fact)
	return classify(tx.Create(&transition).Error)
}

// Insert writes a CREATED execution and its initial transition atomically, without an interview pin.
func (s Store) Insert(ctx context.Context, snap execution.Snapshot, fact execution.TransitionFact) error {
	return s.inTx(ctx, func(tx *gorm.DB) error { return insertExecution(tx, snap, fact) })
}

// InsertInterview writes the shared execution and its transition, then the assignment pin, atomically.
func (s Store) InsertInterview(ctx context.Context, assignmentID string, snap execution.Snapshot, fact execution.TransitionFact) error {
	return s.inTx(ctx, func(tx *gorm.DB) error {
		if err := insertExecution(tx, snap, fact); err != nil {
			return err
		}
		pin := PinModel{ID: snap.ID, AssignmentID: assignmentID, InstanceID: snap.Binding.InstanceID, RevisionID: snap.Binding.RevisionID}
		return classify(tx.Create(&pin).Error)
	})
}

// isTerminal reports whether a stored state is a committed ending.
func isTerminal(state execution.State) bool {
	return state == execution.StateCompleted || state == execution.StateFailed || state == execution.StateCancelled
}

// Update locks the execution row, runs fn on a detached snapshot and persists only a real change.
// Only the lifecycle columns are written, so the instance and revision binding cannot change. Updating
// a changed snapshot that no longer matches a nonterminal row at the locked version reports the locked
// outcome: ErrTerminalConflict for a committed ending, otherwise ErrStaleVersion.
func (s Store) Update(ctx context.Context, id string, fn func(*execution.Snapshot) (*execution.TransitionFact, error)) error {
	return s.inTx(ctx, func(tx *gorm.DB) error {
		var model ExecutionModel
		err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", id).Take(&model).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return executionport.ErrNotFound
		}
		if err != nil {
			return err
		}
		stored, err := toSnapshot(model)
		if err != nil {
			return err
		}
		restored, err := execution.RestoreExecution(stored)
		if err != nil {
			return err
		}
		next := restored.Snapshot()
		fact, err := fn(&next)
		if err != nil {
			return err
		}
		if reflect.DeepEqual(stored, next) {
			return nil
		}
		result := tx.Model(&ExecutionModel{}).
			Where("id = ? AND state_version = ? AND state IN ?", id, stored.StateVersion, nonterminalStates).
			Select("state", "state_version", "committed_progress_ref", "activated_at", "terminal_at", "terminal_reason", "updated_at").
			Updates(updateColumns(next))
		if result.Error != nil {
			return classify(result.Error)
		}
		if result.RowsAffected == 0 {
			if isTerminal(stored.State) {
				return execution.ErrTerminalConflict
			}
			return execution.ErrStaleVersion
		}
		if fact == nil {
			return nil
		}
		transition := fromTransition(id, *fact)
		return classify(tx.Create(&transition).Error)
	})
}

// Find returns the stored execution snapshot by id or ErrNotFound.
func (s Store) Find(ctx context.Context, id string) (execution.Snapshot, error) {
	var model ExecutionModel
	err := s.session(ctx).Where("id = ?", id).Take(&model).Error
	return mapRead(model, err)
}

// FindByAssignment returns the execution linked to the assignment by the interview pin or ErrNotFound.
func (s Store) FindByAssignment(ctx context.Context, assignmentID string) (execution.Snapshot, error) {
	var model ExecutionModel
	err := s.session(ctx).Model(&ExecutionModel{}).
		Select(`"agent_executions".*`).
		Joins("JOIN interview_executions ON interview_executions.id = agent_executions.id").
		Where("interview_executions.assignment_id = ?", assignmentID).
		Take(&model).Error
	return mapRead(model, err)
}

// mapRead translates absence to ErrNotFound and restores a found row through the aggregate.
func mapRead(model ExecutionModel, err error) (execution.Snapshot, error) {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return execution.Snapshot{}, executionport.ErrNotFound
	}
	if err != nil {
		return execution.Snapshot{}, err
	}
	return toSnapshot(model)
}

// WithTransaction runs fn in one transaction that store calls made with the handed context join.
// It joins the transaction already carried by ctx instead of opening a nested one.
func (s Store) WithTransaction(ctx context.Context, fn func(ctx context.Context) error) error {
	return s.inTx(ctx, func(tx *gorm.DB) error {
		return fn(context.WithValue(ctx, txKey{}, tx))
	})
}
