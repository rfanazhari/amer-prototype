package taskusecase_test

import (
	"context"
	"errors"
	"github.com/rfanazhari/amer-prototype/application/usecase/taskusecase"
	"github.com/rfanazhari/amer-prototype/domain/aggregate"
	"github.com/rfanazhari/amer-prototype/domain/valueobject"
	"github.com/rfanazhari/amer-prototype/test/mocks"
	"github.com/rfanazhari/amer-prototype/test/testdata"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"strings"
	"testing"
	"time"
)

type createCase struct {
	name, title                   string
	past, persistFailure, wantErr bool
}

func TestCreate(t *testing.T) {
	tests := []createCase{{"create", "\u2003Prepare report ", false, false, false}, {"empty", "", false, false, true}, {"overlong", strings.Repeat("界", 101), false, false, true}, {"past", "task", true, false, true}, {"persistence failure", "task", false, true, true}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := mocks.NewTask(t)
			clock := mocks.NewClock(t)
			failure := errors.New("database failure")
			if tt.title != "" && len([]rune(strings.TrimSpace(tt.title))) <= 100 {
				clock.On("Now").Return(testdata.Now())
			}
			if !tt.wantErr || tt.persistFailure {
				repo.On("Create", mock.Anything, mock.Anything).Return(func(_ context.Context, task *aggregate.Task) error {
					require.Equal(t, testdata.Now(), task.Snapshot().CreatedAt)
					require.Nil(t, task.Snapshot().CompletedAt)
					if tt.persistFailure {
						return failure
					}
					return nil
				})
			}
			var due *time.Time
			if tt.past {
				date := testdata.Now().Add(-time.Millisecond)
				due = &date
			}
			got, err := (taskusecase.Create{Repository: repo, Clock: clock}).Execute(context.Background(), testdata.ID, tt.title, due)
			if tt.wantErr {
				require.Error(t, err)
				if tt.persistFailure {
					require.ErrorIs(t, err, failure)
				} else if !tt.past {
					require.ErrorIs(t, err, valueobject.ErrInvalidTitle)
				}
				return
			}
			require.NoError(t, err)
			require.Equal(t, "Prepare report", got.Title)
			require.Equal(t, "todo", got.Status)
			require.False(t, got.Overdue)
		})
	}
}
