package integration_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/rfanazhari/amer-prototype/application/repository/agentconfigport"
	"github.com/rfanazhari/amer-prototype/application/usecase/agentbindingusecase"
	aggregate "github.com/rfanazhari/amer-prototype/domain/aggregate/agentconfig"
	"github.com/rfanazhari/amer-prototype/domain/valueobject"
	"github.com/rfanazhari/amer-prototype/infrastructure/repository/agentbindingrepository"
	"github.com/rfanazhari/amer-prototype/infrastructure/repository/agentconfigrepository"
	"github.com/rfanazhari/amer-prototype/test/testdata"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// TestAgentConfigurationBindingConsumers proves pinned assignment and execution survive later publication.
func TestAgentConfigurationBindingConsumers(t *testing.T) {
	store, db := acpStore(t)
	installBindingConsumers(t, db)
	ctx := context.Background()
	account := insertACPAccount(t, db)
	doc := persistDocument(t, store, uuid.NewString(), []string{"total"})
	id := doc.Document.Snapshot().ID
	firstVal := insertOwnedValidation(t, store, id, 1, true)
	firstRev := publishOwned(t, store, id, firstVal, account.ID, uuid.NewString(), testdata.TokenHash)

	assignments := agentbindingrepository.Assignments{DB: db}
	executions := agentbindingrepository.Executions{DB: db}
	pin := agentbindingusecase.PinAssignment{Instances: store, Assignments: assignments}
	assignmentID := uuid.NewString()
	executionID := uuid.NewString()
	assignment, err := pin.Execute(ctx, assignmentID, testdata.CandidateID, id)
	require.NoError(t, err)
	require.Equal(t, firstRev, assignment.RevisionID)

	started, err := (agentbindingusecase.StartExecution{Assignments: assignments, Executions: executions, Revisions: store}).
		Execute(ctx, executionID, assignmentID)
	require.NoError(t, err)
	require.Equal(t, firstRev, started.RevisionID)

	replaceDocumentDraft(t, store, id, 1, []string{"total", "currency"})
	secondVal := insertOwnedValidation(t, store, id, 2, true)
	secondRev := publishOwned(t, store, id, secondVal, account.ID, uuid.NewString(), "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa2")
	require.NotEqual(t, firstRev, secondRev)

	held, err := assignments.Find(ctx, assignmentID)
	require.NoError(t, err)
	require.Equal(t, firstRev, held.RevisionID)
	requirements, err := (agentbindingusecase.LoadRequirements{Executions: executions, Revisions: store}).Execute(ctx, executionID)
	require.NoError(t, err)
	require.Equal(t, firstRev, requirements.RevisionID)
	require.Equal(t, []string{"total"}, requirements.RequiredFields)

	later, err := pin.Execute(ctx, uuid.NewString(), testdata.CandidateID, id)
	require.NoError(t, err)
	require.Equal(t, secondRev, later.RevisionID)
}

// installBindingConsumers creates Interview-shaped assignment and execution pin tables for T11 proof.
func installBindingConsumers(t *testing.T, db *gorm.DB) {
	t.Helper()
	t.Cleanup(func() {
		require.NoError(t, db.Exec("DROP TABLE IF EXISTS interview_executions, interview_assignments CASCADE").Error)
	})
	require.NoError(t, db.Exec(`CREATE TABLE interview_assignments (
		id uuid PRIMARY KEY,
		candidate_id uuid NOT NULL,
		instance_id uuid NOT NULL,
		published_revision_id uuid NOT NULL
	)`).Error)
	require.NoError(t, db.Exec(`CREATE TABLE interview_executions (
		id uuid PRIMARY KEY,
		assignment_id uuid NOT NULL REFERENCES interview_assignments(id) ON DELETE RESTRICT ON UPDATE RESTRICT,
		instance_id uuid NOT NULL,
		published_revision_id uuid NOT NULL
	)`).Error)
}

// replaceDocumentDraft writes the next Document draft version without changing the active revision.
func replaceDocumentDraft(t *testing.T, store agentconfigrepository.Store, instanceID string, expected uint64, fields []string) {
	t.Helper()
	cfg, err := valueobject.NewDocumentConfiguration(fields)
	require.NoError(t, err)
	version, err := aggregate.NewDraftVersion(expected)
	require.NoError(t, err)
	require.NoError(t, store.WithinInstance(context.Background(), instanceID, func(tx agentconfigport.InstanceTx) error {
		current, err := tx.Instance(context.Background())
		if err != nil {
			return err
		}
		if err := current.Document.ReplaceDraft(version, cfg, testdata.Now().Add(2*time.Minute)); err != nil {
			return err
		}
		return tx.SaveInstance(context.Background(), current)
	}))
}
