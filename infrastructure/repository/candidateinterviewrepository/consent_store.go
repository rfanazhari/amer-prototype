// Package candidateinterviewrepository persists candidate-interview consent and controller records through GORM.
package candidateinterviewrepository

import (
	"context"

	"github.com/rfanazhari/amer-prototype/application/repository/candidateinterviewport"
	"github.com/rfanazhari/amer-prototype/application/service"
	"github.com/rfanazhari/amer-prototype/domain/aggregate/candidateinterview"
	"gorm.io/gorm"
)

// ConsentStore implements the consent persistence ports through a runtime-owned GORM pool.
type ConsentStore struct {
	DB    *gorm.DB
	Clock service.Clock
}

// WithAssignment is an inert stub until the assignment boundary is implemented.
func (s ConsentStore) WithAssignment(_ context.Context, _, _ string, _ func(context.Context) error) error {
	return candidateinterviewport.ErrAssignmentUnavailable
}

// Insert is an inert stub until consent storage is implemented.
func (s ConsentStore) Insert(_ context.Context, _ candidateinterview.ConsentSnapshot) error {
	return candidateinterviewport.ErrNotFound
}

// Find is an inert stub until consent reads are implemented.
func (s ConsentStore) Find(_ context.Context, _ string, _ candidateinterview.NoticeVersion) (candidateinterview.ConsentSnapshot, error) {
	return candidateinterview.ConsentSnapshot{}, candidateinterviewport.ErrNotFound
}

// FindByID is an inert stub until consent reads are implemented.
func (s ConsentStore) FindByID(_ context.Context, _, _ string) (candidateinterview.ConsentSnapshot, error) {
	return candidateinterview.ConsentSnapshot{}, candidateinterviewport.ErrNotFound
}

var (
	_ candidateinterviewport.ConsentTransaction = ConsentStore{}
	_ candidateinterviewport.Consents           = ConsentStore{}
)
