package taskusecase

import (
	"context"
	"time"

	"github.com/rfanazhari/amer-prototype/application/dto"
	"github.com/rfanazhari/amer-prototype/application/repository"
	"github.com/rfanazhari/amer-prototype/application/service"
	"github.com/rfanazhari/amer-prototype/domain/aggregate"
	"github.com/rfanazhari/amer-prototype/domain/event"
)

// CompleteOperation defines the complete command contract consumed by delivery.
type CompleteOperation interface {
	// Execute returns detached task details on success and preserves classified failures.
	Execute(context.Context, string) (*dto.Task, error)
}

// Complete coordinates a serialized completion, reserved event capacity and post-commit publication.
type Complete struct {
	Repository repository.Task
	Clock      service.Clock
	Queue      service.EventQueue
}

// Execute reserves capacity inside the transaction callback, cancels on transaction failure and publishes only after a successful commit.
func (u Complete) Execute(ctx context.Context, id string) (*dto.Task, error) {
	now := u.Clock.Now().UTC().Truncate(time.Millisecond)
	var reservation service.Reservation
	var completed event.TaskCompleted
	task, err := u.Repository.Update(ctx, id, func(task *aggregate.Task) error {
		var err error
		completed, err = task.Complete(now)
		if err != nil {
			return err
		}
		reservation, err = u.Queue.Reserve()
		return err
	})
	if err != nil {
		if reservation != nil {
			reservation.Cancel()
		}
		return nil, err
	}
	reservation.Publish(completed)
	return dto.FromTask(task, now), nil
}
