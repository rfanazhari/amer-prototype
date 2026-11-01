package candidateinterviewrepository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/rfanazhari/amer-prototype/application/repository/candidateinterviewport"
	"github.com/rfanazhari/amer-prototype/domain/aggregate/candidateinterview"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// ControllerStore implements the controller persistence port through a runtime-owned GORM pool.
type ControllerStore struct {
	DB *gorm.DB
}

// ControllerModel maps interview_controllers independently of domain and API types.
type ControllerModel struct {
	AssignmentID string    `gorm:"column:assignment_id;type:uuid;primaryKey"`
	ExecutionID  string    `gorm:"column:execution_id;type:uuid;not null"`
	ControllerID string    `gorm:"column:controller_id;type:text;not null"`
	Version      int64     `gorm:"column:version;not null"`
	UpdatedAt    time.Time `gorm:"column:updated_at;not null;autoUpdateTime:false"`
}

// TableName fixes the controller table used by GORM.
func (ControllerModel) TableName() string { return "interview_controllers" }

// Find returns the controller of the assignment, or ErrNotFound.
func (s ControllerStore) Find(ctx context.Context, assignmentID string) (candidateinterview.ControllerSnapshot, error) {
	var model ControllerModel
	err := s.DB.WithContext(ctx).Where("assignment_id = ?", assignmentID).Take(&model).Error
	controller, err := restoreController(model, err)
	if err != nil {
		return candidateinterview.ControllerSnapshot{}, err
	}
	return controller.Snapshot(), nil
}

// Claim stores the first controller and maps a unique violation onto ErrAlreadyExists.
func (s ControllerStore) Claim(ctx context.Context, snap candidateinterview.ControllerSnapshot) error {
	model := controllerModel(snap)
	if err := s.DB.WithContext(ctx).Create(&model).Error; err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return candidateinterviewport.ErrAlreadyExists
		}
		return err
	}
	return nil
}

// Update locks the controller row, runs fn on the restored aggregate and writes id, version and updated_at explicitly.
func (s ControllerStore) Update(ctx context.Context, assignmentID string, fn func(*candidateinterview.InterviewController) error) error {
	return s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var model ControllerModel
		err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("assignment_id = ?", assignmentID).Take(&model).Error
		controller, err := restoreController(model, err)
		if err != nil {
			return err
		}
		if err := fn(controller); err != nil {
			return err
		}
		next := controllerModel(controller.Snapshot())
		return tx.Model(&ControllerModel{}).Where("assignment_id = ?", assignmentID).
			Select("controller_id", "version", "updated_at").Updates(&next).Error
	})
}

// controllerModel maps a domain snapshot onto the storage model with UTC time.
func controllerModel(snap candidateinterview.ControllerSnapshot) ControllerModel {
	return ControllerModel{
		AssignmentID: snap.AssignmentID, ExecutionID: snap.ExecutionID, ControllerID: snap.ControllerID.String(),
		Version: snap.Version, UpdatedAt: snap.UpdatedAt.UTC(),
	}
}

// restoreController translates a read result into an aggregate; a corrupt stored row is a technical error.
func restoreController(model ControllerModel, err error) (*candidateinterview.InterviewController, error) {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, candidateinterviewport.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	id, err := candidateinterview.ParseControllerID(model.ControllerID)
	if err != nil {
		return nil, fmt.Errorf("stored controller of assignment %s: %w", model.AssignmentID, err)
	}
	controller, err := candidateinterview.RestoreController(candidateinterview.ControllerSnapshot{
		AssignmentID: model.AssignmentID, ExecutionID: model.ExecutionID, ControllerID: id,
		Version: model.Version, UpdatedAt: model.UpdatedAt.UTC(),
	})
	if err != nil {
		return nil, fmt.Errorf("stored controller of assignment %s: %w", model.AssignmentID, err)
	}
	return controller, nil
}

var _ candidateinterviewport.Controllers = ControllerStore{}
