package integration_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/rfanazhari/amer-prototype/application/repository/candidateinterviewport"
	"github.com/rfanazhari/amer-prototype/application/repository/executionport"
	"github.com/rfanazhari/amer-prototype/domain/aggregate/candidateinterview"
	"github.com/rfanazhari/amer-prototype/infrastructure/repository/candidateinterviewrepository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// controllerFixture holds one assignment with a real execution pin for the controller foreign key.
type controllerFixture struct {
	executionFixture
	executionID string
	store       candidateinterviewrepository.ControllerStore
}

// newControllerFixture pins one execution to the release-schema assignment.
func newControllerFixture(t *testing.T) controllerFixture {
	t.Helper()
	f := newExecutionFixture(t)
	snap, fact := f.admit(t)
	require.NoError(t, f.store.WithAssignment(context.Background(), f.assignmentID, func(ctx context.Context, _ executionport.AssignmentRow) error {
		return f.store.InsertInterview(ctx, f.assignmentID, snap, fact)
	}))
	t.Cleanup(func() {
		require.NoError(t, f.db.Exec(`DELETE FROM interview_controllers WHERE assignment_id = ?`, f.assignmentID).Error)
	})
	return controllerFixture{executionFixture: f, executionID: snap.ID, store: candidateinterviewrepository.ControllerStore{DB: f.db}}
}

// claim stores the first controller for the fixture at version 1.
func (f controllerFixture) claim(t *testing.T, tab string) candidateinterview.ControllerSnapshot {
	t.Helper()
	id, err := candidateinterview.ParseControllerID(tab)
	require.NoError(t, err)
	c, err := candidateinterview.ClaimController(f.assignmentID, f.executionID, id, time.Now().UTC().Truncate(time.Millisecond))
	require.NoError(t, err)
	require.NoError(t, f.store.Claim(context.Background(), c.Snapshot()))
	return c.Snapshot()
}

// takeover replaces the controller with tab when the stored version equals expected.
func (f controllerFixture) takeover(tab string, expected int64) error {
	next, err := candidateinterview.ParseControllerID(tab)
	if err != nil {
		return err
	}
	return f.store.Update(context.Background(), f.assignmentID, func(c *candidateinterview.InterviewController) error {
		return c.Takeover(expected, next, false, time.Now().UTC().Truncate(time.Millisecond))
	})
}

// TestCandidateControllerRepositoryPostgres proves the single row, explicit update, stale refusal and one concurrent winner.
func TestCandidateControllerRepositoryPostgres(t *testing.T) {
	t.Run("one controller row per assignment", func(t *testing.T) {
		f := newControllerFixture(t)
		first := f.claim(t, "tab-a")
		id, err := candidateinterview.ParseControllerID("tab-z")
		require.NoError(t, err)
		again, err := candidateinterview.ClaimController(f.assignmentID, f.executionID, id, time.Now().UTC())
		require.NoError(t, err)
		require.ErrorIs(t, f.store.Claim(context.Background(), again.Snapshot()), candidateinterviewport.ErrAlreadyExists)
		got, err := f.store.Find(context.Background(), f.assignmentID)
		require.NoError(t, err)
		assert.Equal(t, "tab-a", got.ControllerID.String())
		assert.EqualValues(t, 1, got.Version)
		assert.True(t, first.UpdatedAt.Equal(got.UpdatedAt))
		assert.Equal(t, int64(1), f.count(t, `SELECT count(*) FROM interview_controllers WHERE assignment_id = ?`, f.assignmentID))
	})
	t.Run("absent controller", func(t *testing.T) {
		f := newControllerFixture(t)
		_, err := f.store.Find(context.Background(), f.assignmentID)
		require.ErrorIs(t, err, candidateinterviewport.ErrNotFound)
		require.ErrorIs(t, f.takeover("tab-b", 1), candidateinterviewport.ErrNotFound)
	})
	t.Run("takeover writes id, version and updated_at", func(t *testing.T) {
		f := newControllerFixture(t)
		before := f.claim(t, "tab-a")
		require.NoError(t, f.takeover("tab-b", 1))
		got, err := f.store.Find(context.Background(), f.assignmentID)
		require.NoError(t, err)
		assert.Equal(t, "tab-b", got.ControllerID.String())
		assert.EqualValues(t, 2, got.Version)
		assert.True(t, got.UpdatedAt.After(before.UpdatedAt) || got.UpdatedAt.Equal(before.UpdatedAt))
		assert.Equal(t, f.executionID, got.ExecutionID)
	})
	t.Run("stale expected version keeps the previous id", func(t *testing.T) {
		f := newControllerFixture(t)
		f.claim(t, "tab-a")
		require.NoError(t, f.takeover("tab-b", 1))
		require.ErrorIs(t, f.takeover("tab-c", 1), candidateinterview.ErrStaleControllerVersion)
		got, err := f.store.Find(context.Background(), f.assignmentID)
		require.NoError(t, err)
		assert.Equal(t, "tab-b", got.ControllerID.String())
		assert.EqualValues(t, 2, got.Version)
	})
	t.Run("concurrent takeovers with one expected version have one winner", func(t *testing.T) {
		f := newControllerFixture(t)
		f.claim(t, "tab-a")
		const callers = 8
		var wg sync.WaitGroup
		errs := make([]error, callers)
		tabs := []string{"tab-1", "tab-2", "tab-3", "tab-4", "tab-5", "tab-6", "tab-7", "tab-8"}
		for i := 0; i < callers; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				errs[i] = f.takeover(tabs[i], 1)
			}()
		}
		wg.Wait()
		winners := 0
		for _, err := range errs {
			if err == nil {
				winners++
				continue
			}
			require.ErrorIs(t, err, candidateinterview.ErrStaleControllerVersion)
		}
		assert.Equal(t, 1, winners)
		got, err := f.store.Find(context.Background(), f.assignmentID)
		require.NoError(t, err)
		assert.EqualValues(t, 2, got.Version)
		assert.NotEqual(t, "tab-a", got.ControllerID.String())
	})
	t.Run("interview_executions gains no column", func(t *testing.T) {
		f := newControllerFixture(t)
		assert.Equal(t, int64(0), f.count(t, `SELECT count(*) FROM information_schema.columns WHERE table_name = 'interview_executions' AND column_name IN ('controller_id', 'controller_version')`))
	})
}
