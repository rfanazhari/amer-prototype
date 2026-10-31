package taskusecase_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/rfanazhari/amer-prototype/application/repository"
	"github.com/rfanazhari/amer-prototype/application/service"
	"github.com/rfanazhari/amer-prototype/application/usecase/taskusecase"
	"github.com/rfanazhari/amer-prototype/domain/aggregate"
	"github.com/rfanazhari/amer-prototype/domain/enum"
	"github.com/rfanazhari/amer-prototype/domain/event"
	"github.com/rfanazhari/amer-prototype/domain/valueobject"
	"github.com/rfanazhari/amer-prototype/test/mocks"
	"github.com/rfanazhari/amer-prototype/test/testdata"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// TestCompleteCommitFailureCancelsReservation checks that a reported commit failure releases capacity without publishing.
func TestCompleteCommitFailureCancelsReservation(t *testing.T) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	title, err := valueobject.NewTitle("task")
	require.NoError(t, err)
	task, err := aggregate.NewTask("id", title, nil, now)
	require.NoError(t, err)
	require.NoError(t, task.Start())
	repo := mocks.NewTask(t)
	clock := mocks.NewClock(t)
	queue := mocks.NewEventQueue(t)
	reservation := mocks.NewReservation(t)
	clock.On("Now").Return(now)
	queue.On("Reserve").Return(reservation, nil)
	reservation.On("Cancel").Return()
	failure := errors.New("commit failed")
	repo.On("Update", mock.Anything, "id", mock.Anything).Return(func(_ context.Context, _ string, change func(*aggregate.Task) error) (*aggregate.Task, error) {
		require.NoError(t, change(task))
		return nil, failure
	})
	operation := taskusecase.Complete{Repository: repo, Clock: clock, Queue: queue}
	_, err = operation.Execute(context.Background(), "id")
	require.ErrorIs(t, err, failure)
	reservation.AssertNotCalled(t, "Publish", mock.Anything)
}

// TestCompletePublishesAfterCommit checks that publication occurs only after reported transaction success.
func TestCompletePublishesAfterCommit(t *testing.T) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	title, err := valueobject.NewTitle("task")
	require.NoError(t, err)
	task, err := aggregate.NewTask("id", title, nil, now)
	require.NoError(t, err)
	require.NoError(t, task.Start())
	repo := mocks.NewTask(t)
	clock := mocks.NewClock(t)
	queue := mocks.NewEventQueue(t)
	reservation := mocks.NewReservation(t)
	clock.On("Now").Return(now)
	queue.On("Reserve").Return(reservation, nil)
	committed := false
	repo.On("Update", mock.Anything, "id", mock.Anything).Return(func(_ context.Context, _ string, change func(*aggregate.Task) error) (*aggregate.Task, error) {
		require.NoError(t, change(task))
		committed = true
		return task, nil
	})
	reservation.On("Publish", event.TaskCompleted{TaskID: "id", CompletedAt: now}).Run(func(mock.Arguments) { require.True(t, committed) }).Return()
	operation := taskusecase.Complete{Repository: repo, Clock: clock, Queue: queue}
	_, err = operation.Execute(context.Background(), "id")
	require.NoError(t, err)
}

// completionFailureCase defines a rejected completion and its persistence/publication expectations.
type completionFailureCase struct {
	name                  string
	state                 enum.TaskState
	queueFailure, missing bool
	wantErr               bool
}

// TestCompleteRejectsBeforePublication checks forbidden transitions, missing records and queue refusal without persisted mutation.
func TestCompleteRejectsBeforePublication(t *testing.T) {
	tests := []completionFailureCase{{"todo", enum.Todo, false, false, true}, {"already done", enum.Done, false, false, true}, {"missing", enum.InProgress, false, true, true}, {"queue full", enum.InProgress, true, false, true}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			task := testdata.Task(t, tt.state, nil)
			repo := mocks.NewTask(t)
			clock := mocks.NewClock(t)
			queue := mocks.NewEventQueue(t)
			clock.On("Now").Return(testdata.Now().Add(time.Hour))
			if tt.queueFailure {
				queue.On("Reserve").Return(nil, service.ErrEventUnavailable)
			}
			repo.On("Update", mock.Anything, testdata.ID, mock.Anything).Return(func(_ context.Context, _ string, change func(*aggregate.Task) error) (*aggregate.Task, error) {
				if tt.missing {
					return nil, repository.ErrNotFound
				}
				loaded, err := aggregate.RestoreTask(task.Snapshot())
				require.NoError(t, err)
				if err = change(loaded); err != nil {
					return nil, err
				}
				return loaded, nil
			})
			before := task.Snapshot()
			_, err := (taskusecase.Complete{Repository: repo, Clock: clock, Queue: queue}).Execute(context.Background(), testdata.ID)
			require.Error(t, err)
			require.Equal(t, before, task.Snapshot())
			if tt.queueFailure {
				require.ErrorIs(t, err, service.ErrEventUnavailable)
			} else if tt.missing {
				require.ErrorIs(t, err, repository.ErrNotFound)
			} else {
				require.ErrorIs(t, err, aggregate.ErrTransition)
			}
		})
	}
}
