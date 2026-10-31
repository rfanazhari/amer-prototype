package taskrepository

import (
	"context"
	"errors"
	"time"

	"github.com/rfanazhari/amer-prototype/application/repository"
	"github.com/rfanazhari/amer-prototype/domain/aggregate"
	"github.com/rfanazhari/amer-prototype/domain/enum"
	"github.com/rfanazhari/amer-prototype/domain/valueobject"
	"github.com/rfanazhari/amer-prototype/pkg/errs"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type Model struct {
	ID          string `gorm:"type:uuid;primaryKey"`
	Title       string `gorm:"not null"`
	Status      string `gorm:"not null;check:task_status_check,status IN ('todo','in_progress','done')"`
	DueDate     *time.Time
	CreatedAt   time.Time `gorm:"not null;autoCreateTime:false"`
	UpdatedAt   time.Time `gorm:"not null"`
	CompletedAt *time.Time
}

func (Model) TableName() string { return "tasks" }

type Store struct{ DB *gorm.DB }

func utcPointer(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}
	canonical := value.UTC()
	return &canonical
}

func fromTask(task *aggregate.Task) Model {
	s := task.Snapshot()
	return Model{ID: s.ID, Title: s.Title.String(), Status: s.Status.String(), DueDate: s.DueDate, CreatedAt: s.CreatedAt, CompletedAt: s.CompletedAt}
}
func toTask(m Model) (*aggregate.Task, error) {
	title, err := valueobject.NewTitle(m.Title)
	if err != nil {
		return nil, errs.Technical{Cause: err}
	}
	if title.String() != m.Title {
		return nil, errs.Technical{Cause: aggregate.ErrInvalidTask}
	}
	task, err := aggregate.RestoreTask(aggregate.Snapshot{ID: m.ID, Title: title, Status: enum.TaskState(m.Status), DueDate: utcPointer(m.DueDate), CreatedAt: m.CreatedAt.UTC(), CompletedAt: utcPointer(m.CompletedAt)})
	if err != nil {
		return nil, errs.Technical{Cause: err}
	}
	return task, nil
}
func (s Store) Create(ctx context.Context, task *aggregate.Task) error {
	model := fromTask(task)
	return s.DB.WithContext(ctx).Create(&model).Error
}
func (s Store) Find(ctx context.Context, id string) (*aggregate.Task, error) {
	var model Model
	err := s.DB.WithContext(ctx).First(&model, "id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, repository.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return toTask(model)
}
func (s Store) Update(ctx context.Context, id string, change func(*aggregate.Task) error) (*aggregate.Task, error) {
	var result *aggregate.Task
	err := s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var model Model
		err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&model, "id = ?", id).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return repository.ErrNotFound
		}
		if err != nil {
			return err
		}
		result, err = toTask(model)
		if err != nil {
			return err
		}
		if err = change(result); err != nil {
			return err
		}
		updated := fromTask(result)
		return tx.Model(&model).Select("status", "due_date", "completed_at", "updated_at").Updates(updated).Error
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}
