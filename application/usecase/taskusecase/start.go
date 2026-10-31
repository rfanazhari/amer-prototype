package taskusecase

import (
	"context"
	"time"

	"github.com/rfanazhari/amer-prototype/application/dto"
	"github.com/rfanazhari/amer-prototype/application/repository"
	"github.com/rfanazhari/amer-prototype/application/service"
	"github.com/rfanazhari/amer-prototype/domain/aggregate"
)

// StartOperation defines the start command contract consumed by delivery.
type StartOperation interface {
	// Execute returns detached task details on success and preserves classified failures.
	Execute(context.Context, string) (*dto.Task, error)
}

// Start requests a serialized Todo-to-InProgress transition and returns detached details.
type Start struct {
	Repository repository.Task
	Clock      service.Clock
}

// Execute applies Start under repository serialization and observes the resulting details with the injected clock.
func (u Start) Execute(ctx context.Context, id string) (*dto.Task, error) {
	task, err := u.Repository.Update(ctx, id, func(task *aggregate.Task) error { return task.Start() })
	if err != nil {
		return nil, err
	}
	return dto.FromTask(task, u.Clock.Now().UTC().Truncate(time.Millisecond)), nil
}
