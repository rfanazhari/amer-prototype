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
)

type startCase struct {
	name    string
	state   enum.TaskState
	missing bool
	wantErr bool
}

func TestStart(t *testing.T) {
	tests := []startCase{{"todo", enum.Todo, false, false}, {"already started", enum.InProgress, false, true}, {"done", enum.Done, false, true}, {"missing", enum.Todo, true, true}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			task := testdata.Task(t, tt.state, nil)
			repo := mocks.NewTask(t)
			clock := mocks.NewClock(t)
			repo.On("Update", mock.Anything, testdata.ID, mock.Anything).Return(func(_ context.Context, _ string, change func(*aggregate.Task) error) (*aggregate.Task, error) {
				if tt.missing {
					return nil, repository.ErrNotFound
				}
				if err := change(task); err != nil {
					return nil, err
				}
				return task, nil
			})
			if !tt.wantErr {
				clock.On("Now").Return(testdata.Now())
			}
			got, err := (taskusecase.Start{Repository: repo, Clock: clock}).Execute(context.Background(), testdata.ID)
			if tt.wantErr {
				require.Error(t, err)
				if tt.missing {
					require.ErrorIs(t, err, repository.ErrNotFound)
				} else {
					require.ErrorIs(t, err, aggregate.ErrTransition)
				}
				return
			}
			require.NoError(t, err)
			require.Equal(t, "in_progress", got.Status)
		})
	}
}
