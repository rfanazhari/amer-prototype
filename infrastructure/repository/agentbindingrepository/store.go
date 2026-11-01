package agentbindingrepository

import (
	"context"
	"errors"

	"github.com/rfanazhari/amer-prototype/application/repository/agentbindingport"
	"gorm.io/gorm"
)

// Assignments persists Interview-shaped assignment revision pins.
type Assignments struct {
	DB *gorm.DB
}

// Executions persists Interview-shaped execution revision pins.
type Executions struct {
	DB *gorm.DB
}

// AssignmentModel maps interview_assignments independently of domain and API types.
type AssignmentModel struct {
	ID          string `gorm:"type:uuid;primaryKey"`
	CandidateID string `gorm:"column:candidate_id;type:uuid;not null"`
	InstanceID  string `gorm:"column:instance_id;type:uuid;not null"`
	RevisionID  string `gorm:"column:published_revision_id;type:uuid;not null"`
}

// TableName fixes the Interview assignment pin table used by GORM.
func (AssignmentModel) TableName() string { return "interview_assignments" }

// ExecutionModel maps interview_executions independently of domain and API types.
type ExecutionModel struct {
	ID           string `gorm:"type:uuid;primaryKey"`
	AssignmentID string `gorm:"column:assignment_id;type:uuid;not null"`
	InstanceID   string `gorm:"column:instance_id;type:uuid;not null"`
	RevisionID   string `gorm:"column:published_revision_id;type:uuid;not null"`
}

// TableName fixes the Interview execution pin table used by GORM.
func (ExecutionModel) TableName() string { return "interview_executions" }

// Insert writes one assignment pin to an already-published revision.
func (s Assignments) Insert(ctx context.Context, binding agentbindingport.AssignmentBinding) error {
	return s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return tx.Create(&AssignmentModel{
			ID: binding.AssignmentID, CandidateID: binding.CandidateID, InstanceID: binding.InstanceID, RevisionID: binding.RevisionID,
		}).Error
	})
}

// Find restores one assignment pin or returns ErrNotFound.
func (s Assignments) Find(ctx context.Context, assignmentID string) (agentbindingport.AssignmentBinding, error) {
	var model AssignmentModel
	if err := s.DB.WithContext(ctx).Where("id = ?", assignmentID).Take(&model).Error; err != nil {
		return agentbindingport.AssignmentBinding{}, notFoundOrError(err)
	}
	return agentbindingport.AssignmentBinding{
		AssignmentID: model.ID, CandidateID: model.CandidateID, InstanceID: model.InstanceID, RevisionID: model.RevisionID,
	}, nil
}

// Insert writes one execution pin copied from its assignment at start.
func (s Executions) Insert(ctx context.Context, binding agentbindingport.ExecutionBinding) error {
	return s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return tx.Create(&ExecutionModel{
			ID: binding.ExecutionID, AssignmentID: binding.AssignmentID, InstanceID: binding.InstanceID, RevisionID: binding.RevisionID,
		}).Error
	})
}

// Find restores one execution pin or returns ErrNotFound.
func (s Executions) Find(ctx context.Context, executionID string) (agentbindingport.ExecutionBinding, error) {
	var model ExecutionModel
	if err := s.DB.WithContext(ctx).Where("id = ?", executionID).Take(&model).Error; err != nil {
		return agentbindingport.ExecutionBinding{}, notFoundOrError(err)
	}
	return agentbindingport.ExecutionBinding{
		ExecutionID: model.ID, AssignmentID: model.AssignmentID, InstanceID: model.InstanceID, RevisionID: model.RevisionID,
	}, nil
}

// notFoundOrError translates GORM absence into ErrNotFound while preserving other storage failures.
func notFoundOrError(err error) error {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return agentbindingport.ErrNotFound
	}
	return err
}
