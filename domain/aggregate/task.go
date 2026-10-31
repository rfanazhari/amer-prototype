package aggregate

import (
	"errors"
	"time"

	"github.com/rfanazhari/amer-prototype/domain/enum"
	"github.com/rfanazhari/amer-prototype/domain/event"
	"github.com/rfanazhari/amer-prototype/domain/valueobject"
)

var (
	ErrTransition  = errors.New("invalid transition")
	ErrDueDate     = errors.New("invalid due date")
	ErrInvalidTask = errors.New("invalid task")
)

type Snapshot struct {
	ID          string
	Title       valueobject.Title
	Status      enum.TaskState
	DueDate     *time.Time
	CreatedAt   time.Time
	CompletedAt *time.Time
}

type Task struct{ state Snapshot }

func copyTime(t *time.Time) *time.Time {
	if t == nil {
		return nil
	}
	v := *t
	return &v
}

func NewTask(id string, title valueobject.Title, due *time.Time, now time.Time) (*Task, error) {
	if id == "" || !title.Valid() || now.IsZero() {
		return nil, ErrInvalidTask
	}
	if due != nil && due.Before(now) {
		return nil, ErrDueDate
	}
	return &Task{state: Snapshot{ID: id, Title: title, Status: enum.Todo, DueDate: copyTime(due), CreatedAt: now}}, nil
}

func RestoreTask(s Snapshot) (*Task, error) {
	if s.ID == "" || !s.Title.Valid() || s.CreatedAt.IsZero() {
		return nil, ErrInvalidTask
	}
	if !s.Status.Valid() {
		return nil, ErrInvalidTask
	}
	if (s.Status == enum.Done) != (s.CompletedAt != nil) || (s.CompletedAt != nil && s.CompletedAt.Before(s.CreatedAt)) {
		return nil, ErrInvalidTask
	}
	s.DueDate = copyTime(s.DueDate)
	s.CompletedAt = copyTime(s.CompletedAt)
	return &Task{state: s}, nil
}

func (t *Task) Snapshot() Snapshot {
	s := t.state
	s.DueDate = copyTime(s.DueDate)
	s.CompletedAt = copyTime(s.CompletedAt)
	return s
}
func (t *Task) Start() error {
	if t == nil || !t.state.Status.Valid() || t.state.ID == "" {
		return ErrInvalidTask
	}
	if t.state.Status != enum.Todo {
		return ErrTransition
	}
	t.state.Status = enum.InProgress
	return nil
}
func (t *Task) Complete(now time.Time) (event.TaskCompleted, error) {
	if t == nil || !t.state.Status.Valid() || t.state.ID == "" {
		return event.TaskCompleted{}, ErrInvalidTask
	}
	if t.state.Status != enum.InProgress {
		return event.TaskCompleted{}, ErrTransition
	}
	if now.IsZero() || now.Before(t.state.CreatedAt) {
		return event.TaskCompleted{}, ErrInvalidTask
	}
	t.state.Status = enum.Done
	t.state.CompletedAt = copyTime(&now)
	return event.TaskCompleted{TaskID: t.state.ID, CompletedAt: now}, nil
}
func (t *Task) Reschedule(due *time.Time, now time.Time) error {
	if t == nil || !t.state.Status.Valid() || t.state.ID == "" {
		return ErrInvalidTask
	}
	if t.state.Status == enum.Done {
		return ErrTransition
	}
	if due != nil && due.Before(now) {
		return ErrDueDate
	}
	t.state.DueDate = copyTime(due)
	return nil
}
func (t *Task) Overdue(now time.Time) bool {
	return t.state.Status != enum.Done && t.state.DueDate != nil && now.After(*t.state.DueDate)
}
