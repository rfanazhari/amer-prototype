package interviewrepository

import (
	"context"
	"errors"
	"time"

	"github.com/rfanazhari/amer-prototype/application/repository/interviewport"
	"github.com/rfanazhari/amer-prototype/domain/aggregate/agentconfig"
	"github.com/rfanazhari/amer-prototype/domain/aggregate/execution"
	"github.com/rfanazhari/amer-prototype/domain/interview"
	"github.com/rfanazhari/amer-prototype/domain/valueobject"
	"github.com/rfanazhari/amer-prototype/infrastructure/database/transaction"
	"github.com/rfanazhari/amer-prototype/infrastructure/repository/agentconfigrepository"
	"gorm.io/gorm"
)

// Revisions reads only the execution's immutable revision, never the instance's active publication.
type Revisions struct{ DB *gorm.DB }

// pinRow holds the immutable binding and trusted lifecycle activation instant.
type pinRow struct {
	InstanceID  string
	RevisionID  string
	ActivatedAt *time.Time
}

// binding reads an execution pin whose generic and interview binding agree.
func binding(ctx context.Context, db *gorm.DB, id string) (pinRow, error) {
	var row pinRow
	err := transaction.Session(ctx, db).Table("interview_executions AS pin").Select("pin.instance_id, pin.published_revision_id AS revision_id, run.activated_at").Joins("JOIN agent_executions AS run ON run.id = pin.id AND run.instance_id = pin.instance_id AND run.published_revision_id = pin.published_revision_id").Where("pin.id = ?", id).Take(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return row, interviewport.ErrNotFound
	}
	return row, err
}

// Find restores the HR revision pinned at execution admission.
func (s Revisions) Find(ctx context.Context, id string) (agentconfig.PublishedRevision[valueobject.HRConfiguration], error) {
	pin, err := binding(ctx, s.DB, id)
	if err != nil {
		return agentconfig.PublishedRevision[valueobject.HRConfiguration]{}, err
	}
	stored, err := (agentconfigrepository.Store{DB: transaction.Session(ctx, s.DB)}).FindRevision(ctx, pin.InstanceID, pin.RevisionID)
	if err != nil {
		return agentconfig.PublishedRevision[valueobject.HRConfiguration]{}, err
	}
	if stored.HR == nil {
		return agentconfig.PublishedRevision[valueobject.HRConfiguration]{}, interview.ErrRestore
	}
	return *stored.HR, nil
}

// Pins supplies immutable configuration and the original trusted running instant to restoration.
type Pins struct{ DB *gorm.DB }

// Find returns independently owned pin facts from the lifecycle and immutable revision.
func (s Pins) Find(ctx context.Context, id string) (interviewport.PinnedContext, error) {
	pin, err := binding(ctx, s.DB, id)
	if err != nil {
		return interviewport.PinnedContext{}, err
	}
	if pin.ActivatedAt == nil {
		return interviewport.PinnedContext{}, interview.ErrRestore
	}
	revision, err := Revisions(s).Find(ctx, id)
	if err != nil {
		return interviewport.PinnedContext{}, err
	}
	return interviewport.PinnedContext{RevisionID: pin.RevisionID, Configuration: revision.Snapshot().Configuration, RunningAt: pin.ActivatedAt.UTC()}, nil
}

// IsCommitted checks that the exact opaque progress reference names a sealed closing handoff.
func (s Store) IsCommitted(ctx context.Context, id string, ref execution.ProgressRef) (bool, error) {
	if string(ref) != id {
		return false, nil
	}
	record, err := s.Find(ctx, id)
	if err != nil {
		return false, err
	}
	p, err := interview.RestoreProgress(record.Snapshot)
	if err != nil {
		return false, err
	}
	return p.ReadyForCompletion(), nil
}
