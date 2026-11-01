package dto_test

import (
	"testing"
	"time"

	"github.com/rfanazhari/amer-prototype/application/dto"
	"github.com/rfanazhari/amer-prototype/domain/aggregate/execution"
	"github.com/stretchr/testify/require"
)

// TestNewExecutionViewKeepsNullTimesNullAndCopiesPointers covers CREATED and COMPLETED shapes.
func TestNewExecutionViewKeepsNullTimesNullAndCopiesPointers(t *testing.T) {
	created := time.Date(2026, 3, 4, 5, 0, 0, 0, time.UTC)
	e, snap, _, err := execution.NewExecution("exec-1", execution.Binding{InstanceID: "inst", RevisionID: "rev"}, created)
	require.NoError(t, err)

	view := dto.NewExecutionView(snap, "asg-1")
	require.NotNil(t, view)
	require.Equal(t, "exec-1", view.ID)
	require.Equal(t, "asg-1", view.AssignmentID)
	require.Equal(t, "CREATED", view.State)
	require.EqualValues(t, 1, view.StateVersion)
	require.Nil(t, view.ActivatedAt)
	require.Nil(t, view.TerminalAt)
	require.Empty(t, view.ProgressRef)
	require.Empty(t, view.TerminalReason)

	_, _, err = e.Activate(1, "a", "op", 1, nil, created.Add(time.Minute))
	require.NoError(t, err)
	done, _, err := e.Complete("prog-1", 2, 1, nil, "a", "op", created.Add(2*time.Minute))
	require.NoError(t, err)
	view = dto.NewExecutionView(done, "")
	require.NotNil(t, view)
	require.Equal(t, "COMPLETED", view.State)
	require.Equal(t, "prog-1", view.ProgressRef)
	require.NotNil(t, view.ActivatedAt)
	require.NotNil(t, view.TerminalAt)
	*view.TerminalAt = created
	require.Equal(t, created.Add(2*time.Minute), *done.TerminalAt)
}
