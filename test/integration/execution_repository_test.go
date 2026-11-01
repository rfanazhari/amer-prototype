package integration_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/rfanazhari/amer-prototype/application/repository/executionport"
	"github.com/rfanazhari/amer-prototype/domain/aggregate/execution"
	"github.com/rfanazhari/amer-prototype/infrastructure/repository/executionrepository"
	"github.com/rfanazhari/amer-prototype/test/testdata"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// executionFixture holds one prepared release-schema graph for execution store proofs.
type executionFixture struct {
	store        executionrepository.Store
	db           *gorm.DB
	binding      execution.Binding
	assignmentID string
}

// errRollbackProbe forces a callback failure after the store has written rows.
var errRollbackProbe = errors.New("rollback probe")

// newExecutionFixture builds an instance, published revision, candidate and assignment on the release schema.
func newExecutionFixture(t *testing.T) executionFixture {
	t.Helper()
	configStore, db := acpStore(t)
	account := insertACPAccount(t, db)
	doc := persistDocument(t, configStore, uuid.NewString(), []string{"total"})
	instanceID := doc.Document.Snapshot().ID
	validation := insertOwnedValidation(t, configStore, instanceID, 1, true)
	revisionID := publishOwned(t, configStore, instanceID, validation, account.ID, uuid.NewString(), testdata.TokenHash)
	candidateID, assignmentID := uuid.NewString(), uuid.NewString()
	require.NoError(t, db.Exec(`INSERT INTO candidates (id, name, email, created_by, created_at, updated_at) VALUES (?, 'Execution Probe', 'probe@example.test', ?, now(), now())`,
		candidateID, account.ID).Error)
	require.NoError(t, db.Exec(`INSERT INTO interview_assignments (id, candidate_id, instance_id, published_revision_id) VALUES (?, ?, ?, ?)`,
		assignmentID, candidateID, instanceID, revisionID).Error)
	t.Cleanup(func() {
		require.NoError(t, db.Exec(`DELETE FROM agent_execution_transitions WHERE execution_id IN (SELECT id FROM agent_executions WHERE instance_id = ?)`, instanceID).Error)
		require.NoError(t, db.Exec(`DELETE FROM interview_executions WHERE assignment_id = ?`, assignmentID).Error)
		require.NoError(t, db.Exec(`DELETE FROM agent_executions WHERE instance_id = ?`, instanceID).Error)
		require.NoError(t, db.Exec(`DELETE FROM interview_assignments WHERE id = ?`, assignmentID).Error)
		require.NoError(t, db.Exec(`DELETE FROM candidates WHERE id = ?`, candidateID).Error)
	})
	return executionFixture{
		store: executionrepository.Store{DB: db}, db: db, assignmentID: assignmentID,
		binding: execution.Binding{InstanceID: instanceID, RevisionID: revisionID},
	}
}

// admit builds a CREATED execution for the fixture binding with a fresh id and an actor on the initial transition.
func (f executionFixture) admit(t *testing.T) (execution.Snapshot, execution.TransitionFact) {
	t.Helper()
	now := time.Now().UTC().Truncate(time.Millisecond)
	_, snap, fact, err := execution.NewExecution(uuid.NewString(), f.binding, now)
	require.NoError(t, err)
	fact.ActorRef = "interview:test"
	return snap, fact
}

// count returns the number of rows a query reports for one id.
func (f executionFixture) count(t *testing.T, query string, args ...any) int64 {
	t.Helper()
	var n int64
	require.NoError(t, f.db.Raw(query, args...).Scan(&n).Error)
	return n
}

// executionStoreCase names one real PostgreSQL execution store proof.
type executionStoreCase struct {
	name   string
	verify func(t *testing.T, f executionFixture)
}

// TestExecutionStorePostgres proves rollback, the unique assignment pin, row locking and backfill restoration.
func TestExecutionStorePostgres(t *testing.T) {
	tests := []executionStoreCase{
		{name: "failed assignment callback rolls back execution and pin", verify: proveExecutionRollback},
		{name: "second execution for one assignment is refused and rolled back", verify: proveUniqueAssignmentPin},
		{name: "update persists completion with its transition and rolls back a failed callback", verify: proveExecutionUpdate},
		{name: "backfilled version-7 pin restores as CREATED", verify: proveBackfilledRoundTrip},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.verify(t, newExecutionFixture(t))
		})
	}
}

// proveExecutionRollback shows a callback error after InsertInterview leaves neither row behind.
func proveExecutionRollback(t *testing.T, f executionFixture) {
	snap, fact := f.admit(t)
	err := f.store.WithAssignment(context.Background(), f.assignmentID, func(ctx context.Context, row executionport.AssignmentRow) error {
		require.Equal(t, f.binding.RevisionID, row.RevisionID)
		require.NoError(t, f.store.InsertInterview(ctx, f.assignmentID, snap, fact))
		return errRollbackProbe
	})
	require.ErrorIs(t, err, errRollbackProbe)
	require.Zero(t, f.count(t, `SELECT count(*) FROM agent_executions WHERE id = ?`, snap.ID))
	require.Zero(t, f.count(t, `SELECT count(*) FROM interview_executions WHERE assignment_id = ?`, f.assignmentID))
	require.Zero(t, f.count(t, `SELECT count(*) FROM agent_execution_transitions WHERE execution_id = ?`, snap.ID))
}

// proveUniqueAssignmentPin shows the unique pin refuses a second execution and removes its shared row.
func proveUniqueAssignmentPin(t *testing.T, f executionFixture) {
	ctx := context.Background()
	first, firstFact := f.admit(t)
	require.NoError(t, f.store.WithAssignment(ctx, f.assignmentID, func(ctx context.Context, _ executionport.AssignmentRow) error {
		return f.store.InsertInterview(ctx, f.assignmentID, first, firstFact)
	}))
	second, secondFact := f.admit(t)
	err := f.store.WithAssignment(ctx, f.assignmentID, func(ctx context.Context, _ executionport.AssignmentRow) error {
		return f.store.InsertInterview(ctx, f.assignmentID, second, secondFact)
	})
	require.ErrorIs(t, err, executionport.ErrAlreadyExists)
	require.EqualValues(t, 1, f.count(t, `SELECT count(*) FROM interview_executions WHERE assignment_id = ?`, f.assignmentID))
	require.Zero(t, f.count(t, `SELECT count(*) FROM agent_executions WHERE id = ?`, second.ID))
	found, err := f.store.FindByAssignment(ctx, f.assignmentID)
	require.NoError(t, err)
	require.Equal(t, first.ID, found.ID)
}

// proveExecutionUpdate shows a locked update persists nulls and history, and a failed callback changes nothing.
func proveExecutionUpdate(t *testing.T, f executionFixture) {
	ctx := context.Background()
	snap, fact := f.admit(t)
	require.NoError(t, f.store.Insert(ctx, snap, fact))
	activate := func(s *execution.Snapshot) (*execution.TransitionFact, error) {
		agg, err := execution.RestoreExecution(*s)
		require.NoError(t, err)
		next, transition, err := agg.Activate(1, "runtime:test", "op-1", 1, nil, snap.CreatedAt.Add(time.Second))
		if err != nil {
			return nil, err
		}
		*s = next
		return &transition, nil
	}
	require.ErrorIs(t, f.store.Update(ctx, snap.ID, func(s *execution.Snapshot) (*execution.TransitionFact, error) {
		_, err := activate(s)
		require.NoError(t, err)
		return nil, errRollbackProbe
	}), errRollbackProbe)
	stored, err := f.store.Find(ctx, snap.ID)
	require.NoError(t, err)
	require.Equal(t, execution.StateCreated, stored.State)

	require.NoError(t, f.store.Update(ctx, snap.ID, activate))
	progress := uuid.NewString()
	require.NoError(t, f.store.Update(ctx, snap.ID, func(s *execution.Snapshot) (*execution.TransitionFact, error) {
		agg, err := execution.RestoreExecution(*s)
		require.NoError(t, err)
		next, transition, err := agg.Complete(execution.ProgressRef(progress), 2, 1, nil, "runtime:test", "op-1", snap.CreatedAt.Add(2*time.Second))
		if err != nil {
			return nil, err
		}
		*s = next
		return &transition, nil
	}))
	done, err := f.store.Find(ctx, snap.ID)
	require.NoError(t, err)
	require.Equal(t, execution.StateCompleted, done.State)
	require.EqualValues(t, 3, done.StateVersion)
	require.NotNil(t, done.ProgressRef)
	require.Equal(t, progress, string(*done.ProgressRef))
	require.Equal(t, f.binding, done.Binding)
	require.EqualValues(t, 3, f.count(t, `SELECT count(*) FROM agent_execution_transitions WHERE execution_id = ?`, snap.ID))

	err = f.store.Update(ctx, snap.ID, func(s *execution.Snapshot) (*execution.TransitionFact, error) {
		s.State, s.TerminalReason = execution.StateCancelled, "late_cancel"
		return nil, nil
	})
	require.ErrorIs(t, err, execution.ErrTerminalConflict)
	again, err := f.store.Find(ctx, snap.ID)
	require.NoError(t, err)
	require.Equal(t, execution.StateCompleted, again.State)
}

// proveBackfilledRoundTrip applies the migration's backfill statement to a pin and restores it as CREATED.
func proveBackfilledRoundTrip(t *testing.T, f executionFixture) {
	pinID := uuid.NewString()
	require.NoError(t, f.db.Exec(`ALTER TABLE interview_executions DROP CONSTRAINT interview_executions_id_fkey`).Error)
	t.Cleanup(func() {
		require.NoError(t, f.db.Exec(`DELETE FROM interview_executions WHERE id = ?`, pinID).Error)
		require.NoError(t, f.db.Exec(`DELETE FROM agent_executions WHERE id = ?`, pinID).Error)
		require.NoError(t, f.db.Exec(`ALTER TABLE interview_executions ADD CONSTRAINT interview_executions_id_fkey FOREIGN KEY (id) REFERENCES public.agent_executions(id) ON UPDATE RESTRICT ON DELETE RESTRICT`).Error)
	})
	require.NoError(t, f.db.Exec(`INSERT INTO interview_executions (id, assignment_id, instance_id, published_revision_id) VALUES (?, ?, ?, ?)`,
		pinID, f.assignmentID, f.binding.InstanceID, f.binding.RevisionID).Error)
	require.NoError(t, f.db.Exec(`INSERT INTO public.agent_executions (id, instance_id, published_revision_id, state, state_version, created_at, updated_at)
SELECT id, instance_id, published_revision_id, 'CREATED', 1, created_at, created_at FROM public.interview_executions WHERE id = ?`, pinID).Error)

	byID, err := f.store.Find(context.Background(), pinID)
	require.NoError(t, err)
	require.Equal(t, execution.StateCreated, byID.State)
	require.EqualValues(t, 1, byID.StateVersion)
	require.Equal(t, f.binding, byID.Binding)
	require.Nil(t, byID.ActivatedAt)
	require.Nil(t, byID.TerminalAt)
	require.Nil(t, byID.ProgressRef)
	require.Equal(t, byID.CreatedAt, byID.UpdatedAt)
	byAssignment, err := f.store.FindByAssignment(context.Background(), f.assignmentID)
	require.NoError(t, err)
	require.Equal(t, byID, byAssignment)
}
