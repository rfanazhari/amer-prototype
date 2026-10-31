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

type CreateOperation interface {
	Execute(context.Context, string, string, *time.Time) (*dto.Task, error)
}

type Create struct {
	Repository repository.Task
	Clock      service.Clock
}

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
