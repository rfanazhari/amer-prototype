package taskquery

import (
	"context"
	"github.com/rfanazhari/amer-prototype/application/dto"
	"github.com/rfanazhari/amer-prototype/application/repository"
	"github.com/rfanazhari/amer-prototype/application/service"
	"time"
)

type Detail struct {
	Repository repository.Task
	Clock      service.Clock
}

func (q Detail) Execute(ctx context.Context, id string) (*dto.Task, error) {
	task, err := q.Repository.Find(ctx, id)
	if err != nil {
		return nil, err
	}
	return dto.FromTask(task, q.Clock.Now().UTC().Truncate(time.Millisecond)), nil
}

type DetailOperation interface {
	Execute(context.Context, string) (*dto.Task, error)
}
