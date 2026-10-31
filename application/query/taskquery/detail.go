package taskquery

import (
	"context"
	"github.com/rfanazhari/amer-prototype/application/dto"
	"github.com/rfanazhari/amer-prototype/application/repository"
	"github.com/rfanazhari/amer-prototype/application/service"
	"time"
)

// Detail reads a task and evaluates its read result using the injected observation clock.
type Detail struct {
	Repository repository.Task
	Clock      service.Clock
}

// Execute loads a task without mutation and returns its detached details, preserving repository failures.
func (q Detail) Execute(ctx context.Context, id string) (*dto.Task, error) {
	task, err := q.Repository.Find(ctx, id)
	if err != nil {
		return nil, err
	}
	return dto.FromTask(task, q.Clock.Now().UTC().Truncate(time.Millisecond)), nil
}

// DetailOperation is the task-details query contract consumed by delivery.
type DetailOperation interface {
	// Execute returns detached task details on success and preserves classified failures.
	Execute(context.Context, string) (*dto.Task, error)
}
