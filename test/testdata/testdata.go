package testdata

import (
	"github.com/rfanazhari/amer-prototype/domain/aggregate"
	"github.com/rfanazhari/amer-prototype/domain/enum"
	"github.com/rfanazhari/amer-prototype/domain/valueobject"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

const ID = "9d96c891-8118-46d4-92d2-79da2337faba"

// Now returns the fixed UTC baseline used by public domain and application scenarios.
func Now() time.Time { return time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC) }

// Task returns a fresh valid aggregate in the requested lifecycle state, created through public domain operations.
func Task(t *testing.T, state enum.TaskState, due *time.Time) *aggregate.Task {
	t.Helper()
	title, err := valueobject.NewTitle("Prepare report")
	require.NoError(t, err)
	task, err := aggregate.NewTask(ID, title, due, Now())
	require.NoError(t, err)
	if state == enum.InProgress || state == enum.Done {
		require.NoError(t, task.Start())
	}
	if state == enum.Done {
		_, err = task.Complete(Now().Add(time.Minute))
		require.NoError(t, err)
	}
	return task
}
