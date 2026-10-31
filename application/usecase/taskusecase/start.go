package taskusecase

import (
	"context"
	"time"

	"github.com/rfanazhari/amer-prototype/application/dto"
	"github.com/rfanazhari/amer-prototype/application/repository"
	"github.com/rfanazhari/amer-prototype/application/service"
	"github.com/rfanazhari/amer-prototype/domain/aggregate"
)

type StartOperation interface {
	Execute(context.Context, string) (*dto.Task, error)
}

type Start struct {
	Repository repository.Task
	Clock      service.Clock
}

func (u Start) Execute(ctx context.Context, id string) (*dto.Task, error) {
	task, err := u.Repository.Update(ctx, id, func(task *aggregate.Task) error { return task.Start() })
	if err != nil {
		return nil, err
	}
	return dto.FromTask(task, u.Clock.Now().UTC().Truncate(time.Millisecond)), nil
}
