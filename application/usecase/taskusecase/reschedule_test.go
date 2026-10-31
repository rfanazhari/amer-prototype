package taskusecase_test

import (
	"context"
	"github.com/rfanazhari/amer-prototype/application/repository"
	"github.com/rfanazhari/amer-prototype/application/usecase/taskusecase"
	"github.com/rfanazhari/amer-prototype/domain/aggregate"
	"github.com/rfanazhari/amer-prototype/domain/enum"
	"github.com/rfanazhari/amer-prototype/test/mocks"
	"github.com/rfanazhari/amer-prototype/test/testdata"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

type rescheduleCase struct {
	name                    string
	state                   enum.TaskState
	delta                   time.Duration
	clear, missing, wantErr bool
}

func TestReschedule(t *testing.T) {
	tests := []rescheduleCase{{"equality", enum.Todo, 0, false, false, false}, {"future", enum.InProgress, time.Hour, false, false, false}, {"past", enum.Todo, -time.Millisecond, false, false, true}, {"clear", enum.InProgress, 0, true, false, false}, {"done", enum.Done, time.Hour, false, false, true}, {"missing", enum.Todo, 0, false, true, true}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			initial := testdata.Now().Add(2 * time.Hour)
			task := testdata.Task(t, tt.state, &initial)
			before := task.Snapshot()
			repo := mocks.NewTask(t)
			clock := mocks.NewClock(t)
			clock.On("Now").Return(testdata.Now())
			repo.On("Update", mock.Anything, testdata.ID, mock.Anything).Return(func(_ context.Context, _ string, change func(*aggregate.Task) error) (*aggregate.Task, error) {
				if tt.missing {
					return nil, repository.ErrNotFound
				}
				if err := change(task); err != nil {
					return nil, err
				}
				return task, nil
			})
			candidate := testdata.Now().Add(tt.delta)
			due := &candidate
			if tt.clear {
				due = nil
			}
			got, err := (taskusecase.Reschedule{Repository: repo, Clock: clock}).Execute(context.Background(), testdata.ID, due)
			if tt.wantErr {
				require.Error(t, err)
				require.Equal(t, before, task.Snapshot())
				return
			}
			require.NoError(t, err)
			require.Equal(t, due, got.DueDate)
		})
	}
}
