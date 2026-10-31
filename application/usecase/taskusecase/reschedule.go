package taskusecase

import (
	"context"
	"time"

	"github.com/rfanazhari/amer-prototype/application/dto"
	"github.com/rfanazhari/amer-prototype/application/repository"
	"github.com/rfanazhari/amer-prototype/application/service"
	"github.com/rfanazhari/amer-prototype/domain/aggregate"
)

// RescheduleOperation defines the reschedule command contract consumed by delivery.
type RescheduleOperation interface {
	// Execute returns detached task details on success and preserves classified failures.
	Execute(context.Context, string, *time.Time) (*dto.Task, error)
}

// Reschedule requests a serialized due-date replacement or clearing operation and returns detached details.
type Reschedule struct {
	Repository repository.Task
	Clock      service.Clock
}

// Execute samples trusted time and applies the candidate date inside the serialized repository mutation.
func (u Reschedule) Execute(ctx context.Context, id string, due *time.Time) (*dto.Task, error) {
	now := u.Clock.Now().UTC().Truncate(time.Millisecond)
	task, err := u.Repository.Update(ctx, id, func(task *aggregate.Task) error { return task.Reschedule(due, now) })
	if err != nil {
		return nil, err
	}
	return dto.FromTask(task, now), nil
}
