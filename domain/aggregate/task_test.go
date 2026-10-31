package aggregate_test

import (
	"github.com/rfanazhari/amer-prototype/domain/aggregate"
	"github.com/rfanazhari/amer-prototype/domain/enum"
	"github.com/rfanazhari/amer-prototype/domain/valueobject"
	"github.com/rfanazhari/amer-prototype/test/testdata"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

type transitionCase struct {
	name      string
	state     enum.TaskState
	operation string
	wantErr   bool
}

func TestTaskLifecycle(t *testing.T) {
	tests := []transitionCase{{"start todo", enum.Todo, "start", false}, {"start in progress", enum.InProgress, "start", true}, {"start done", enum.Done, "start", true}, {"complete todo", enum.Todo, "complete", true}, {"complete in progress", enum.InProgress, "complete", false}, {"complete done", enum.Done, "complete", true}, {"reschedule done", enum.Done, "reschedule", true}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			task := testdata.Task(t, tt.state, nil)
			before := task.Snapshot()
			now := testdata.Now().Add(time.Hour)
			var err error
			switch tt.operation {
			case "start":
				err = task.Start()
			case "complete":
				var completedID string
				e, eErr := task.Complete(now)
				err = eErr
				completedID = e.TaskID
				if err == nil {
					require.Equal(t, testdata.ID, completedID)
					require.Equal(t, now, e.CompletedAt)
				}
			case "reschedule":
				err = task.Reschedule(nil, now)
			}
			if tt.wantErr {
				require.ErrorIs(t, err, aggregate.ErrTransition)
				require.Equal(t, before, task.Snapshot())
				return
			}
			require.NoError(t, err)
			require.Equal(t, before.CreatedAt, task.Snapshot().CreatedAt)
			if tt.operation == "complete" {
				require.Equal(t, enum.Done, task.Snapshot().Status)
				require.Equal(t, &now, task.Snapshot().CompletedAt)
			} else {
				require.Equal(t, enum.InProgress, task.Snapshot().Status)
			}
		})
	}
}

type dueCase struct {
	name            string
	delta           time.Duration
	absent, wantErr bool
}

func TestTaskDueDateBoundaries(t *testing.T) {
	tests := []dueCase{{"none", 0, true, false}, {"past", -time.Millisecond, false, true}, {"equality", 0, false, false}, {"future", time.Millisecond, false, false}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			now := testdata.Now()
			due := now.Add(tt.delta)
			candidate := &due
			if tt.absent {
				candidate = nil
			}
			title, err := valueobject.NewTitle("Prepare report")
			require.NoError(t, err)
			task, err := aggregate.NewTask(testdata.ID, title, candidate, now)
			if tt.wantErr {
				require.ErrorIs(t, err, aggregate.ErrDueDate)
			} else {
				require.NoError(t, err)
				require.Equal(t, enum.Todo, task.Snapshot().Status)
				require.Equal(t, now, task.Snapshot().CreatedAt)
				require.Nil(t, task.Snapshot().CompletedAt)
			}
			existing := testdata.Task(t, enum.Todo, nil)
			before := existing.Snapshot()
			err = existing.Reschedule(candidate, now)
			if tt.wantErr {
				require.ErrorIs(t, err, aggregate.ErrDueDate)
				require.Equal(t, before, existing.Snapshot())
			} else {
				require.NoError(t, err)
				require.Equal(t, candidate, existing.Snapshot().DueDate)
			}
		})
	}
}

func TestTaskBoundariesAndRestoration(t *testing.T) {
	now := testdata.Now()
	due := now.Add(time.Hour)
	task := testdata.Task(t, enum.InProgress, &due)
	due = due.Add(time.Hour)
	require.Equal(t, now.Add(time.Hour), *task.Snapshot().DueDate)
	snapshot := task.Snapshot()
	*snapshot.DueDate = now.Add(3 * time.Hour)
	require.Equal(t, now.Add(time.Hour), *task.Snapshot().DueDate)
	require.False(t, task.Overdue(now.Add(time.Hour)))
	require.True(t, task.Overdue(now.Add(time.Hour+time.Millisecond)))
	restored, err := aggregate.RestoreTask(task.Snapshot())
	require.NoError(t, err)
	require.True(t, restored.Overdue(now.Add(2*time.Hour)))
	require.Equal(t, task.Snapshot(), restored.Snapshot())
	require.NoError(t, restored.Reschedule(nil, now.Add(2*time.Hour)))
	require.Nil(t, restored.Snapshot().DueDate)
	require.False(t, restored.Overdue(now.Add(3*time.Hour)))
	completed, err := task.Complete(now.Add(time.Minute))
	require.NoError(t, err)
	require.Equal(t, now.Add(time.Minute), completed.CompletedAt)
	require.False(t, task.Overdue(now.Add(2*time.Hour)))
	finished := task.Snapshot()
	*finished.CompletedAt = now.Add(3 * time.Hour)
	require.Equal(t, now.Add(time.Minute), *task.Snapshot().CompletedAt)
}

type restorationCase struct {
	name    string
	alter   func(*aggregate.Snapshot)
	wantErr bool
}

func TestRestoreRejectsInvalidHistory(t *testing.T) {
	tests := []restorationCase{
		{"empty id", func(s *aggregate.Snapshot) { s.ID = "" }, true},
		{"zero title", func(s *aggregate.Snapshot) { s.Title = valueobject.Title{} }, true},
		{"unknown state", func(s *aggregate.Snapshot) { s.Status = enum.TaskState("unknown") }, true},
		{"missing creation", func(s *aggregate.Snapshot) { s.CreatedAt = time.Time{} }, true},
		{"done without completion", func(s *aggregate.Snapshot) { s.Status = enum.Done }, true},
		{"todo with completion", func(s *aggregate.Snapshot) { now := testdata.Now(); s.CompletedAt = &now }, true},
		{"completion before creation", func(s *aggregate.Snapshot) {
			past := testdata.Now().Add(-time.Millisecond)
			s.Status = enum.Done
			s.CompletedAt = &past
		}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			snapshot := testdata.Task(t, enum.Todo, nil).Snapshot()
			tt.alter(&snapshot)
			_, err := aggregate.RestoreTask(snapshot)
			require.ErrorIs(t, err, aggregate.ErrInvalidTask)
		})
	}
}

func TestZeroTaskCannotMutate(t *testing.T) {
	task := new(aggregate.Task)
	require.ErrorIs(t, task.Reschedule(nil, testdata.Now()), aggregate.ErrInvalidTask)
	require.ErrorIs(t, task.Start(), aggregate.ErrInvalidTask)
	_, err := task.Complete(testdata.Now())
	require.ErrorIs(t, err, aggregate.ErrInvalidTask)
	_, err = aggregate.NewTask(testdata.ID, valueobject.Title{}, nil, testdata.Now())
	require.ErrorIs(t, err, aggregate.ErrInvalidTask)
	title, err := valueobject.NewTitle("task")
	require.NoError(t, err)
	_, err = aggregate.NewTask("", title, nil, testdata.Now())
	require.ErrorIs(t, err, aggregate.ErrInvalidTask)
	task = testdata.Task(t, enum.InProgress, nil)
	before := task.Snapshot()
	_, err = task.Complete(testdata.Now().Add(-time.Millisecond))
	require.ErrorIs(t, err, aggregate.ErrInvalidTask)
	require.Equal(t, before, task.Snapshot())
}
