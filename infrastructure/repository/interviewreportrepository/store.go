package interviewreportrepository

import (
	"context"
	"time"

	"github.com/rfanazhari/amer-prototype/application/repository/interviewreportport"
	"github.com/rfanazhari/amer-prototype/domain/aggregate/interviewreport"
	"gorm.io/gorm"
)

// Store persists interview reports through a shared GORM database pool.
type Store struct{ DB *gorm.DB }

// InTransaction is not implemented yet.
func (s Store) InTransaction(_ context.Context, _ func(ctx context.Context) error) error {
	return nil
}

// CompletedAt is not implemented yet.
func (s Store) CompletedAt(_ context.Context, _, _ string) (*time.Time, error) {
	return nil, nil
}

// FindByExecution is not implemented yet.
func (s Store) FindByExecution(_ context.Context, _ string) (interviewreport.Snapshot, error) {
	return interviewreport.Snapshot{}, nil
}

// Insert is not implemented yet.
func (s Store) Insert(_ context.Context, _ interviewreportport.Admission) error {
	return nil
}

// ClaimJob is not implemented yet.
func (s Store) ClaimJob(_ context.Context, _ time.Time) (interviewreportport.ClaimedJob, error) {
	return interviewreportport.ClaimedJob{}, nil
}

// ClaimOutbox is not implemented yet.
func (s Store) ClaimOutbox(_ context.Context, _ int, _ time.Time) ([]interviewreportport.OutboxMessage, error) {
	return nil, nil
}
