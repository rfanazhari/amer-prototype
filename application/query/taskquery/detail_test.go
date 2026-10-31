package taskquery_test

import (
	"context"
	"github.com/rfanazhari/amer-prototype/application/query/taskquery"
	"github.com/rfanazhari/amer-prototype/application/repository"
	"github.com/rfanazhari/amer-prototype/domain/enum"
	"github.com/rfanazhari/amer-prototype/test/mocks"
	"github.com/rfanazhari/amer-prototype/test/testdata"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

type detailCase struct {
	name                              string
	state                             enum.TaskState
	advance                           time.Duration
	noDate, missing, wantErr, overdue bool
}

func TestDetail(t *testing.T) {
	tests := []detailCase{{name: "exact due date", state: enum.InProgress}, {name: "elapsed", state: enum.InProgress, advance: time.Millisecond, overdue: true}, {name: "done", state: enum.Done, advance: time.Hour}, {name: "no due date", state: enum.Todo, noDate: true, advance: time.Hour}, {name: "missing", state: enum.Todo, missing: true, wantErr: true}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			due := testdata.Now().Add(time.Hour)
			date := &due
			if tt.noDate {
				date = nil
			}
			task := testdata.Task(t, tt.state, date)
			before := task.Snapshot()
			repo := mocks.NewTask(t)
			clock := mocks.NewClock(t)
			if tt.missing {
				repo.On("Find", mock.Anything, testdata.ID).Return(nil, repository.ErrNotFound)
			} else {
				repo.On("Find", mock.Anything, testdata.ID).Return(task, nil)
				clock.On("Now").Return(due.Add(tt.advance))
			}
			got, err := (taskquery.Detail{Repository: repo, Clock: clock}).Execute(context.Background(), testdata.ID)
			if tt.wantErr {
				require.ErrorIs(t, err, repository.ErrNotFound)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.overdue, got.Overdue)
			require.Equal(t, before, task.Snapshot())
			if got.DueDate != nil {
				*got.DueDate = got.DueDate.Add(time.Hour)
				require.Equal(t, before, task.Snapshot())
			}
		})
	}
}
