package agentconfigrepository_test

import (
	"context"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/rfanazhari/amer-prototype/application/repository/agentconfigport"
	aggregate "github.com/rfanazhari/amer-prototype/domain/aggregate/agentconfig"
	"github.com/rfanazhari/amer-prototype/domain/valueobject"
	"github.com/rfanazhari/amer-prototype/infrastructure/repository/agentconfigrepository"
	"github.com/rfanazhari/amer-prototype/pkg/errs"
	"github.com/rfanazhari/amer-prototype/test/testdata"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// persistCase names one configuration SQL behavior that the adapter must perform.
type persistCase struct {
	name string
	run  func(t *testing.T, store agentconfigrepository.Store, mock sqlmock.Sqlmock)
}

// TestConfigurationPersistenceSQL covers uniqueness, rollback, restoration and explicit null/zero writes.
func TestConfigurationPersistenceSQL(t *testing.T) {
	ctx := context.Background()
	emptyHR := storedHR(t, instanceID, 1, mustHR(t, valueobject.HRInput{}), nil)
	valid := storedHR(t, instanceID, 1, mustHR(t, validHRInput()), nil)
	revision := storedHRRevision(t, revisionID, 1, 1, mustHR(t, validHRInput()))
	tests := []persistCase{
		{
			name: "create writes instance and draft without publication",
			run: func(t *testing.T, store agentconfigrepository.Store, mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectExec(`(?i)SET CONSTRAINTS ALL DEFERRED`).WillReturnResult(sqlmock.NewResult(0, 0))
				mock.ExpectExec(`(?i)INSERT INTO ["']?agent_instances["']?`).WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectExec(`(?i)INSERT INTO ["']?agent_drafts["']?`).WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()
				require.NoError(t, store.Create(ctx, valid))
			},
		},
		{
			name: "create callback failure rolls back both writes",
			run: func(t *testing.T, store agentconfigrepository.Store, mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectExec(`(?i)SET CONSTRAINTS ALL DEFERRED`).WillReturnResult(sqlmock.NewResult(0, 0))
				mock.ExpectExec(`(?i)INSERT INTO ["']?agent_instances["']?`).WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectExec(`(?i)INSERT INTO ["']?agent_drafts["']?`).WillReturnError(assert.AnError)
				mock.ExpectRollback()
				require.Error(t, store.Create(ctx, valid))
			},
		},
		{
			name: "replacement inserts a new draft version instead of mutating the locked row",
			run: func(t *testing.T, store agentconfigrepository.Store, mock sqlmock.Sqlmock) {
				replaced := storedHR(t, instanceID, 2, mustHR(t, valueobject.HRInput{}), nil)
				expectLockedInstance(mock, instanceID, 1, false)
				mock.ExpectExec(`(?i)INSERT INTO ["']?agent_drafts["']?`).
					WithArgs(instanceID, int64(2), []byte(emptyHRJSON), testdata.Now(), testdata.Now()).
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectExec(`(?i)UPDATE ["']?agent_instances["']? SET .*current_draft_version`).
					WithArgs(instanceName, catalogTemplateHR(), catalogVersion(), catalogBehavior(), int64(2), nil, testdata.Now(), instanceID).
					WillReturnResult(sqlmock.NewResult(0, 1))
				mock.ExpectCommit()
				err := store.WithinInstance(ctx, instanceID, func(tx agentconfigport.InstanceTx) error {
					current, err := tx.Instance(ctx)
					require.NoError(t, err)
					require.NotNil(t, current.HR)
					return tx.SaveInstance(ctx, replaced)
				})
				require.NoError(t, err)
			},
		},
		{
			name: "save instance writes null active revision and empty configuration json",
			run: func(t *testing.T, store agentconfigrepository.Store, mock sqlmock.Sqlmock) {
				expectLockedInstance(mock, instanceID, 1, false)
				mock.ExpectExec(`(?i)UPDATE ["']?agent_drafts["']? SET .*configuration`).WillReturnResult(sqlmock.NewResult(0, 1))
				mock.ExpectExec(`(?i)UPDATE ["']?agent_instances["']? SET .*active_revision_id`).
					WithArgs(instanceName, catalogTemplateHR(), catalogVersion(), catalogBehavior(), int64(1), nil, testdata.Now(), instanceID).
					WillReturnResult(sqlmock.NewResult(0, 1))
				mock.ExpectCommit()
				err := store.WithinInstance(ctx, instanceID, func(tx agentconfigport.InstanceTx) error {
					return tx.SaveInstance(ctx, emptyHR)
				})
				require.NoError(t, err)
			},
		},
		{
			name: "insert validation writes explicit ineligible false and empty findings",
			run: func(t *testing.T, store agentconfigrepository.Store, mock sqlmock.Sqlmock) {
				expectLockedInstance(mock, instanceID, 1, false)
				mock.ExpectExec(`(?i)INSERT INTO ["']?agent_validations["']?`).
					WithArgs(validationID, instanceID, int64(1), catalogTemplateHR(), catalogVersion(), catalogBehavior(), false, []byte(emptyFindings), testdata.Now()).
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()
				err := store.WithinInstance(ctx, instanceID, func(tx agentconfigport.InstanceTx) error {
					return tx.InsertValidation(ctx, eligibleValidation(t, validationID, instanceID, 1, false))
				})
				require.NoError(t, err)
			},
		},
		{
			name: "duplicate command identity rolls back and classifies unique loser",
			run: func(t *testing.T, store agentconfigrepository.Store, mock sqlmock.Sqlmock) {
				expectLockedInstance(mock, instanceID, 1, false)
				mock.ExpectExec(`(?i)INSERT INTO ["']?agent_revisions["']?`).WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectExec(`(?i)UPDATE ["']?agent_drafts["']? SET .*configuration`).WillReturnResult(sqlmock.NewResult(0, 1))
				mock.ExpectExec(`(?i)UPDATE ["']?agent_instances["']? SET .*active_revision_id`).WillReturnResult(sqlmock.NewResult(0, 1))
				mock.ExpectExec(`(?i)INSERT INTO ["']?agent_publication_commands["']?`).
					WillReturnError(&pgconn.PgError{Code: "23505", ConstraintName: "agent_publication_commands_pkey"})
				mock.ExpectRollback()
				err := store.WithinInstance(ctx, instanceID, func(tx agentconfigport.InstanceTx) error {
					require.NoError(t, tx.InsertRevision(ctx, revision))
					require.NoError(t, tx.SaveInstance(ctx, storedHR(t, instanceID, 1, mustHR(t, validHRInput()), &aggregate.RevisionIdentity{
						ID: revisionID, InstanceID: instanceID, Number: 1, DraftVersion: mustVersion(t, 1), Template: hrTemplate(),
					})))
					return tx.InsertCommand(ctx, commandIdentity(), agentconfigport.CommandResult{
						Fingerprint: fingerprint(), InstanceID: instanceID, RevisionID: revisionID,
					})
				})
				require.ErrorIs(t, err, agentconfigport.ErrDuplicateCommand)
			},
		},
		{
			name: "unique loser resolves command in a fresh transaction",
			run: func(t *testing.T, store agentconfigrepository.Store, mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"scope", "command_kind", "request_key", "intent_fingerprint", "resulting_revision_id", "instance_id"}).
					AddRow(testdata.ID, commandKind, publishKey, fingerprint(), revisionID, instanceID)
				mock.ExpectQuery(`(?i)SELECT .* FROM ["']?agent_publication_commands["']?`).
					WithArgs(testdata.ID, commandKind, publishKey, 1).
					WillReturnRows(rows)
				got, found, err := store.FindCommand(ctx, commandIdentity())
				require.NoError(t, err)
				require.True(t, found)
				require.Equal(t, fingerprint(), got.Fingerprint)
				require.Equal(t, instanceID, got.InstanceID)
				require.Equal(t, revisionID, got.RevisionID)
			},
		},
		{
			name: "revision insert failure rolls back without command or active pointer",
			run: func(t *testing.T, store agentconfigrepository.Store, mock sqlmock.Sqlmock) {
				expectLockedInstance(mock, instanceID, 1, false)
				mock.ExpectExec(`(?i)INSERT INTO ["']?agent_revisions["']?`).WillReturnError(assert.AnError)
				mock.ExpectRollback()
				err := store.WithinInstance(ctx, instanceID, func(tx agentconfigport.InstanceTx) error {
					return tx.InsertRevision(ctx, revision)
				})
				require.ErrorIs(t, err, assert.AnError)
			},
		},
		{
			name: "active pointer update failure rolls back revision insert",
			run: func(t *testing.T, store agentconfigrepository.Store, mock sqlmock.Sqlmock) {
				expectLockedInstance(mock, instanceID, 1, false)
				mock.ExpectExec(`(?i)INSERT INTO ["']?agent_revisions["']?`).WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectExec(`(?i)UPDATE ["']?agent_drafts["']? SET .*configuration`).WillReturnResult(sqlmock.NewResult(0, 1))
				mock.ExpectExec(`(?i)UPDATE ["']?agent_instances["']? SET .*active_revision_id`).WillReturnError(assert.AnError)
				mock.ExpectRollback()
				err := store.WithinInstance(ctx, instanceID, func(tx agentconfigport.InstanceTx) error {
					require.NoError(t, tx.InsertRevision(ctx, revision))
					return tx.SaveInstance(ctx, storedHR(t, instanceID, 1, mustHR(t, validHRInput()), &aggregate.RevisionIdentity{
						ID: revisionID, InstanceID: instanceID, Number: 1, DraftVersion: mustVersion(t, 1), Template: hrTemplate(),
					}))
				})
				require.ErrorIs(t, err, assert.AnError)
			},
		},
		{
			name: "commit failure after successful writes is uncertain",
			run: func(t *testing.T, store agentconfigrepository.Store, mock sqlmock.Sqlmock) {
				expectLockedInstance(mock, instanceID, 1, false)
				mock.ExpectExec(`(?i)INSERT INTO ["']?agent_revisions["']?`).WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectExec(`(?i)UPDATE ["']?agent_drafts["']? SET .*configuration`).WillReturnResult(sqlmock.NewResult(0, 1))
				mock.ExpectExec(`(?i)UPDATE ["']?agent_instances["']? SET .*active_revision_id`).WillReturnResult(sqlmock.NewResult(0, 1))
				mock.ExpectExec(`(?i)INSERT INTO ["']?agent_publication_commands["']?`).WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit().WillReturnError(assert.AnError)
				err := store.WithinInstance(ctx, instanceID, func(tx agentconfigport.InstanceTx) error {
					require.NoError(t, tx.InsertRevision(ctx, revision))
					require.NoError(t, tx.SaveInstance(ctx, storedHR(t, instanceID, 1, mustHR(t, validHRInput()), &aggregate.RevisionIdentity{
						ID: revisionID, InstanceID: instanceID, Number: 1, DraftVersion: mustVersion(t, 1), Template: hrTemplate(),
					})))
					return tx.InsertCommand(ctx, commandIdentity(), agentconfigport.CommandResult{
						Fingerprint: fingerprint(), InstanceID: instanceID, RevisionID: revisionID,
					})
				})
				var uncertain *agentconfigport.CommitUncertainError
				require.ErrorAs(t, err, &uncertain)
			},
		},
		{
			name: "cross-instance validation is not found",
			run: func(t *testing.T, store agentconfigrepository.Store, mock sqlmock.Sqlmock) {
				expectLockedInstance(mock, instanceID, 1, false)
				rows := sqlmock.NewRows([]string{"id", "instance_id", "draft_version", "template_id", "template_version", "behavior_version", "eligible", "findings", "created_at"}).
					AddRow(validationID, otherInstanceID, 1, catalogTemplateHR(), catalogVersion(), catalogBehavior(), true, []byte(emptyFindings), testdata.Now())
				mock.ExpectQuery(`(?i)SELECT .* FROM ["']?agent_validations["']?`).WillReturnRows(rows)
				mock.ExpectRollback()
				err := store.WithinInstance(ctx, instanceID, func(tx agentconfigport.InstanceTx) error {
					_, err := tx.FindValidation(ctx, validationID)
					return err
				})
				require.ErrorIs(t, err, agentconfigport.ErrNotFound)
			},
		},
		{
			name: "corrupt stored template is technical failure",
			run: func(t *testing.T, store agentconfigrepository.Store, mock sqlmock.Sqlmock) {
				instanceRows := sqlmock.NewRows([]string{"id", "name", "template_id", "template_version", "behavior_version", "current_draft_version", "active_revision_id", "created_at", "updated_at"}).
					AddRow(instanceID, instanceName, "unknown-template", catalogVersion(), catalogBehavior(), 1, nil, testdata.Now(), testdata.Now())
				draftRows := sqlmock.NewRows([]string{"instance_id", "version", "configuration", "created_at", "updated_at"}).
					AddRow(instanceID, 1, []byte(emptyHRJSON), testdata.Now(), testdata.Now())
				mock.ExpectQuery(`(?i)SELECT .* FROM ["']?agent_instances["']?`).WillReturnRows(instanceRows)
				mock.ExpectQuery(`(?i)SELECT .* FROM ["']?agent_drafts["']?`).WillReturnRows(draftRows)
				_, err := store.FindInstance(ctx, instanceID)
				var technical errs.Technical
				require.ErrorAs(t, err, &technical)
			},
		},
		{
			name: "missing instance is not found",
			run: func(t *testing.T, store agentconfigrepository.Store, mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`(?i)SELECT .* FROM ["']?agent_instances["']?`).WillReturnError(gorm.ErrRecordNotFound)
				_, err := store.FindInstance(ctx, instanceID)
				require.ErrorIs(t, err, agentconfigport.ErrNotFound)
			},
		},
		{
			name: "find revision is bounded to one owned row",
			run: func(t *testing.T, store agentconfigrepository.Store, mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id", "instance_id", "revision_number", "source_draft_version", "validation_id", "template_id", "template_version", "behavior_version", "configuration", "created_at"}).
					AddRow(revisionID, instanceID, 1, 1, validationID, catalogTemplateHR(), catalogVersion(), catalogBehavior(), []byte(emptyHRJSON), testdata.Now())
				mock.ExpectQuery(`(?i)SELECT .* FROM ["']?agent_revisions["']?`).
					WithArgs(instanceID, revisionID, 1).
					WillReturnRows(rows)
				got, err := store.FindRevision(ctx, instanceID, revisionID)
				require.NoError(t, err)
				require.NotNil(t, got.HR)
				require.Equal(t, revisionID, got.HR.Snapshot().Identity.ID)
				require.Nil(t, got.Document)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store, mock := sqlStore(t)
			tt.run(t, store, mock)
		})
	}
}

// catalogTemplateHR returns the stored HR template identity.
func catalogTemplateHR() string { return "hr-interviewer" }

// catalogVersion returns the stored template version token.
func catalogVersion() string { return "1" }

// catalogBehavior returns the stored behavior version token.
func catalogBehavior() string { return "1" }

// expectLockedInstance sets the FOR UPDATE instance load and matching current draft row.
func expectLockedInstance(mock sqlmock.Sqlmock, id string, version int64, withActive bool) {
	active := interface{}(nil)
	if withActive {
		active = revisionID
	}
	instanceRows := sqlmock.NewRows([]string{"id", "name", "template_id", "template_version", "behavior_version", "current_draft_version", "active_revision_id", "created_at", "updated_at"}).
		AddRow(id, instanceName, catalogTemplateHR(), catalogVersion(), catalogBehavior(), version, active, testdata.Now(), testdata.Now())
	draftRows := sqlmock.NewRows([]string{"instance_id", "version", "configuration", "created_at", "updated_at"}).
		AddRow(id, version, []byte(emptyHRJSON), testdata.Now(), testdata.Now())
	mock.ExpectBegin()
	mock.ExpectExec(`(?i)SET CONSTRAINTS ALL DEFERRED`).WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectQuery(`(?i)SELECT .* FROM ["']?agent_instances["']? .*FOR UPDATE`).WillReturnRows(instanceRows)
	mock.ExpectQuery(`(?i)SELECT .* FROM ["']?agent_drafts["']?`).WillReturnRows(draftRows)
}
