package taskusecase

import (
	"context"
	"time"

	"github.com/rfanazhari/amer-prototype/application/dto"
	"github.com/rfanazhari/amer-prototype/application/repository"
	"github.com/rfanazhari/amer-prototype/application/service"
	"github.com/rfanazhari/amer-prototype/domain/aggregate"
)

type RescheduleOperation interface {
	Execute(context.Context, string, *time.Time) (*dto.Task, error)
}

type Reschedule struct {
	Repository repository.Task
	Clock      service.Clock
}

func (u Reschedule) Execute(ctx context.Context, id string, due *time.Time) (*dto.Task, error) {
	now := u.Clock.Now().UTC().Truncate(time.Millisecond)
	task, err := u.Repository.Update(ctx, id, func(task *aggregate.Task) error { return task.Reschedule(due, now) })
	if err != nil {
		return nil, err
	}
	return dto.FromTask(task, now), nil
}
