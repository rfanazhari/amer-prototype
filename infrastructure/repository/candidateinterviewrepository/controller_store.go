package candidateinterviewrepository

import (
	"context"
	"errors"
	"time"

	"github.com/rfanazhari/amer-prototype/application/repository/candidateinterviewport"
	"github.com/rfanazhari/amer-prototype/domain/aggregate/candidateinterview"
	"gorm.io/gorm"
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

// errControllerStoreUnimplemented marks the adapter methods that are not written yet.
var errControllerStoreUnimplemented = errors.New("controller store is not implemented")

// Find returns the controller of the assignment, or ErrNotFound.
func (ControllerStore) Find(context.Context, string) (candidateinterview.ControllerSnapshot, error) {
	return candidateinterview.ControllerSnapshot{}, errControllerStoreUnimplemented
}

// Claim stores the first controller, or returns ErrAlreadyExists when the assignment has one.
func (ControllerStore) Claim(context.Context, candidateinterview.ControllerSnapshot) error {
	return errControllerStoreUnimplemented
}

// Update locks the controller row and runs fn on the restored aggregate, then writes the changed fields.
func (ControllerStore) Update(context.Context, string, func(*candidateinterview.InterviewController) error) error {
	return errControllerStoreUnimplemented
}

var _ candidateinterviewport.Controllers = ControllerStore{}
