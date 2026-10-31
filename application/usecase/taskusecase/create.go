package taskusecase

import (
	"context"
	"time"

	"github.com/rfanazhari/amer-prototype/application/dto"

	"github.com/rfanazhari/amer-prototype/application/repository"
	"github.com/rfanazhari/amer-prototype/application/service"
	"github.com/rfanazhari/amer-prototype/domain/aggregate"
	"github.com/rfanazhari/amer-prototype/domain/valueobject"
)

// CreateOperation defines the create command contract consumed by delivery.
type CreateOperation interface {
	// Execute returns detached task details on success and preserves classified failures.
	Execute(context.Context, string, string, *time.Time) (*dto.Task, error)
}

// Create validates a title and creation due date, persists a new Todo task and returns detached details.
type Create struct {
	Repository repository.Task
	Clock      service.Clock
}

// Execute samples trusted time, requests validated domain creation and returns details only after persistence succeeds.
func (u Create) Execute(ctx context.Context, id, input string, due *time.Time) (*dto.Task, error) {
	title, err := valueobject.NewTitle(input)
	if err != nil {
		return nil, err
	}
	now := u.Clock.Now().UTC().Truncate(time.Millisecond)
	task, err := aggregate.NewTask(id, title, due, now)
	if err != nil {
		return nil, err
	}
	if err := u.Repository.Create(ctx, task); err != nil {
		return nil, err
	}
	return dto.FromTask(task, now), nil
}
