package dto

import (
	"github.com/rfanazhari/amer-prototype/domain/aggregate"
	"time"
)

// Task is the detached application read result, including overdue evaluated at the operation observation time.
type Task struct {
	ID          string
	Title       string
	Status      string
	DueDate     *time.Time
	Overdue     bool
	CreatedAt   time.Time
	CompletedAt *time.Time
}

// FromTask copies aggregate facts and evaluates overdue using the supplied trusted observation instant.
func FromTask(task *aggregate.Task, now time.Time) *Task {
	s := task.Snapshot()
	return &Task{ID: s.ID, Title: s.Title.String(), Status: s.Status.String(), DueDate: s.DueDate, Overdue: task.Overdue(now), CreatedAt: s.CreatedAt, CompletedAt: s.CompletedAt}
}
