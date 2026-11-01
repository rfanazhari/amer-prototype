package integration_test

import (
	"context"
	"errors"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/rfanazhari/amer-prototype/application/repository/accessport"
	"github.com/rfanazhari/amer-prototype/application/repository/agentconfigport"
	catalog "github.com/rfanazhari/amer-prototype/domain/agentconfig"
	aggregate "github.com/rfanazhari/amer-prototype/domain/aggregate/agentconfig"
	"github.com/rfanazhari/amer-prototype/domain/enum"
	"github.com/rfanazhari/amer-prototype/domain/valueobject"
	"github.com/rfanazhari/amer-prototype/infrastructure/repository/accessrepository"
	"github.com/rfanazhari/amer-prototype/infrastructure/repository/agentconfigrepository"
	"github.com/rfanazhari/amer-prototype/pkg/errs"
	"github.com/rfanazhari/amer-prototype/test/testdata"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

const (
	acpCommandKind = "publish-agent-configuration/v1"
	acpPublishKey  = "publish-invoice-007"
)

// acpStore opens an authorized integration database prepared at configuration version 3.
func acpStore(t *testing.T) (agentconfigrepository.Store, *gorm.DB) {
	t.Helper()
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL is required")
	}
	db, err := gorm.Open(postgres.Open(url), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	require.NoError(t, err)
	pool, err := db.DB()
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, pool.Close()) })
	prepareSchema(t, db)
	return agentconfigrepository.Store{DB: db}, db
}

// acpCase names one real PostgreSQL publication-repository proof.
type acpCase struct {
	name   string
	verify func(t *testing.T, store agentconfigrepository.Store, db *gorm.DB)
}

// TestAgentConfigurationPublicationRepository proves version-3 races, rollback, uniqueness and restoration.
func TestAgentConfigurationPublicationRepository(t *testing.T) {
	tests := []acpCase{
		{name: "create replace validate publish read for both templates", verify: proveBothTemplates},
		{name: "exact-version replacement race", verify: proveReplacementRace},
		{name: "validation racing replacement", verify: proveValidationRace},
		{name: "publication racing replacement", verify: provePublicationRace},
		{name: "duplicate keys same and different intent", verify: proveCommandIdentity},
		{name: "cross-instance constraint checks", verify: proveCrossInstance},
		{name: "write and commit rollback points", verify: proveRollbackPoints},
		{name: "replay after new active revision", verify: proveReplayAfterLaterPublication},
		{name: "corrupted restoration is technical", verify: proveCorruptRestoration},
		{name: "null and zero mapping plus restart retention", verify: proveNullZeroAndRestart},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store, db := acpStore(t)
			tt.verify(t, store, db)
		})
	}
}

// proveBothTemplates creates, replaces, validates, publishes and reads each catalog template.
func proveBothTemplates(t *testing.T, store agentconfigrepository.Store, db *gorm.DB) {
	ctx := context.Background()
	account := insertACPAccount(t, db)
	hr := persistHR(t, store, uuid.NewString(), mustACPValidHR(t), 1)
	hrID := hr.HR.Snapshot().ID
	hrVal := insertOwnedValidation(t, store, hrID, 1, true)
	hrRev := publishOwned(t, store, hrID, hrVal, account.ID, uuid.NewString(), testdata.TokenHash)
	loadedHR, err := store.FindRevision(ctx, hrID, hrRev)
	require.NoError(t, err)
	require.Equal(t, hrRev, loadedHR.HR.Snapshot().Identity.ID)

	doc := persistDocument(t, store, uuid.NewString(), []string{"total"})
	docID := doc.Document.Snapshot().ID
	docVal := insertOwnedValidation(t, store, docID, 1, true)
	docRev := publishOwned(t, store, docID, docVal, account.ID, uuid.NewString(), "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa1")
	loadedDoc, err := store.FindRevision(ctx, docID, docRev)
	require.NoError(t, err)
	require.Equal(t, docRev, loadedDoc.Document.Snapshot().Identity.ID)
}

// proveReplacementRace serializes two writers so one replacement wins and the other is stale.
func proveReplacementRace(t *testing.T, store agentconfigrepository.Store, _ *gorm.DB) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	instance := persistHR(t, store, uuid.NewString(), mustACPValidHR(t), 1)
	id := instance.HR.Snapshot().ID
	results := make(chan error, 2)
	var workers sync.WaitGroup
	for i := 0; i < 2; i++ {
		workers.Add(1)
		go func() {
			defer workers.Done()
			results <- store.WithinInstance(ctx, id, func(tx agentconfigport.InstanceTx) error {
				current, err := tx.Instance(ctx)
				if err != nil {
					return err
				}
				expected, err := aggregate.NewDraftVersion(1)
				if err != nil {
					return err
				}
				if err := current.HR.ReplaceDraft(expected, mustACPEmptyHR(t), testdata.Now().Add(time.Minute)); err != nil {
					return err
				}
				return tx.SaveInstance(ctx, current)
			})
		}()
	}
	workers.Wait()
	close(results)
	var stale, ok int
	for err := range results {
		if err == nil {
			ok++
			continue
		}
		require.ErrorIs(t, err, aggregate.ErrStale)
		stale++
	}
	require.Equal(t, 1, ok)
	require.Equal(t, 1, stale)
	loaded, err := store.FindInstance(ctx, id)
	require.NoError(t, err)
	require.Equal(t, uint64(2), loaded.HR.Snapshot().DraftVersion.Number())
}

// proveValidationRace rejects inserting an assessment after the locked draft advanced.
func proveValidationRace(t *testing.T, store agentconfigrepository.Store, _ *gorm.DB) {
	ctx := context.Background()
	instance := persistHR(t, store, uuid.NewString(), mustACPValidHR(t), 1)
	id := instance.HR.Snapshot().ID
	require.NoError(t, store.WithinInstance(ctx, id, func(tx agentconfigport.InstanceTx) error {
		current, err := tx.Instance(ctx)
		if err != nil {
			return err
		}
		expected, err := aggregate.NewDraftVersion(1)
		if err != nil {
			return err
		}
		if err := current.HR.ReplaceDraft(expected, mustACPEmptyHR(t), testdata.Now().Add(time.Minute)); err != nil {
			return err
		}
		return tx.SaveInstance(ctx, current)
	}))
	err := store.WithinInstance(ctx, id, func(tx agentconfigport.InstanceTx) error {
		current, err := tx.Instance(ctx)
		if err != nil {
			return err
		}
		if current.HR.Snapshot().DraftVersion.Number() != 1 {
			return aggregate.ErrStale
		}
		return tx.InsertValidation(ctx, mustACPValidation(t, uuid.NewString(), id, 1, true))
	})
	require.ErrorIs(t, err, aggregate.ErrStale)
}

// provePublicationRace refuses publication after the current draft no longer matches the assessment.
func provePublicationRace(t *testing.T, store agentconfigrepository.Store, db *gorm.DB) {
	ctx := context.Background()
	account := insertACPAccount(t, db)
	instance := persistHR(t, store, uuid.NewString(), mustACPValidHR(t), 1)
	id := instance.HR.Snapshot().ID
	validation := insertOwnedValidation(t, store, id, 1, true)
	require.NoError(t, store.WithinInstance(ctx, id, func(tx agentconfigport.InstanceTx) error {
		current, err := tx.Instance(ctx)
		if err != nil {
			return err
		}
		expected, err := aggregate.NewDraftVersion(1)
		if err != nil {
			return err
		}
		if err := current.HR.ReplaceDraft(expected, mustACPEmptyHR(t), testdata.Now().Add(time.Minute)); err != nil {
			return err
		}
		return tx.SaveInstance(ctx, current)
	}))
	err := store.WithinInstance(ctx, id, func(tx agentconfigport.InstanceTx) error {
		current, err := tx.Instance(ctx)
		if err != nil {
			return err
		}
		loaded, err := tx.FindValidation(ctx, validation)
		if err != nil {
			return err
		}
		_, err = current.HR.Publish(uuid.NewString(), loaded, testdata.Now().Add(2*time.Minute))
		return err
	})
	require.ErrorIs(t, err, aggregate.ErrValidation)
	_, _ = account.ID, db
}

// proveCommandIdentity keeps same-intent replay and rejects a different intent on the same scoped key.
func proveCommandIdentity(t *testing.T, store agentconfigrepository.Store, db *gorm.DB) {
	ctx := context.Background()
	account := insertACPAccount(t, db)
	first := persistHR(t, store, uuid.NewString(), mustACPValidHR(t), 1)
	firstID := first.HR.Snapshot().ID
	firstVal := insertOwnedValidation(t, store, firstID, 1, true)
	revision := publishOwned(t, store, firstID, firstVal, account.ID, acpPublishKey, testdata.TokenHash)
	err := store.WithinInstance(ctx, firstID, func(tx agentconfigport.InstanceTx) error {
		return tx.InsertCommand(ctx, agentconfigport.CommandIdentity{Scope: account.ID, Kind: acpCommandKind, Key: acpPublishKey}, agentconfigport.CommandResult{
			Fingerprint: testdata.TokenHash, InstanceID: firstID, RevisionID: revision,
		})
	})
	require.ErrorIs(t, err, agentconfigport.ErrDuplicateCommand)
	got, found, err := store.FindCommand(ctx, agentconfigport.CommandIdentity{Scope: account.ID, Kind: acpCommandKind, Key: acpPublishKey})
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, revision, got.RevisionID)
	require.Equal(t, testdata.TokenHash, got.Fingerprint)

	second := persistHR(t, store, uuid.NewString(), mustACPValidHR(t), 1)
	secondID := second.HR.Snapshot().ID
	secondVal := insertOwnedValidation(t, store, secondID, 1, true)
	err = store.WithinInstance(ctx, secondID, func(tx agentconfigport.InstanceTx) error {
		current, err := tx.Instance(ctx)
		if err != nil {
			return err
		}
		loaded, err := tx.FindValidation(ctx, secondVal)
		if err != nil {
			return err
		}
		published, err := current.HR.Publish(uuid.NewString(), loaded, testdata.Now().Add(time.Minute))
		if err != nil {
			return err
		}
		if err := tx.InsertRevision(ctx, agentconfigport.StoredRevision{HR: &published}); err != nil {
			return err
		}
		if err := tx.SaveInstance(ctx, current); err != nil {
			return err
		}
		return tx.InsertCommand(ctx, agentconfigport.CommandIdentity{Scope: account.ID, Kind: acpCommandKind, Key: acpPublishKey}, agentconfigport.CommandResult{
			Fingerprint: "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb2", InstanceID: secondID, RevisionID: published.Snapshot().Identity.ID,
		})
	})
	require.ErrorIs(t, err, agentconfigport.ErrDuplicateCommand)
}

// proveCrossInstance hides foreign validations and revisions behind not-found.
func proveCrossInstance(t *testing.T, store agentconfigrepository.Store, db *gorm.DB) {
	ctx := context.Background()
	account := insertACPAccount(t, db)
	first := persistHR(t, store, uuid.NewString(), mustACPValidHR(t), 1)
	second := persistHR(t, store, uuid.NewString(), mustACPValidHR(t), 1)
	firstID := first.HR.Snapshot().ID
	secondID := second.HR.Snapshot().ID
	firstVal := insertOwnedValidation(t, store, firstID, 1, true)
	firstRev := publishOwned(t, store, firstID, firstVal, account.ID, uuid.NewString(), testdata.TokenHash)
	err := store.WithinInstance(ctx, secondID, func(tx agentconfigport.InstanceTx) error {
		_, err := tx.FindValidation(ctx, firstVal)
		return err
	})
	require.ErrorIs(t, err, agentconfigport.ErrNotFound)
	_, err = store.FindRevision(ctx, secondID, firstRev)
	require.ErrorIs(t, err, agentconfigport.ErrNotFound)
}

// proveRollbackPoints uses a deferred constraint trigger to leave no partial publication outcome.
func proveRollbackPoints(t *testing.T, store agentconfigrepository.Store, db *gorm.DB) {
	ctx := context.Background()
	account := insertACPAccount(t, db)
	instance := persistHR(t, store, uuid.NewString(), mustACPValidHR(t), 1)
	id := instance.HR.Snapshot().ID
	validation := insertOwnedValidation(t, store, id, 1, true)
	name := "acp_" + uuid.NewString()[:8]
	require.NoError(t, db.Exec("CREATE FUNCTION "+name+"() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'verification commit rejected'; END; $$").Error)
	t.Cleanup(func() { require.NoError(t, db.Exec("DROP FUNCTION IF EXISTS "+name+"() CASCADE").Error) })
	require.NoError(t, db.Exec("CREATE CONSTRAINT TRIGGER "+name+" AFTER INSERT ON agent_publication_commands DEFERRABLE INITIALLY DEFERRED FOR EACH ROW EXECUTE FUNCTION "+name+"()").Error)
	err := store.WithinInstance(ctx, id, func(tx agentconfigport.InstanceTx) error {
		current, err := tx.Instance(ctx)
		if err != nil {
			return err
		}
		loaded, err := tx.FindValidation(ctx, validation)
		if err != nil {
			return err
		}
		published, err := current.HR.Publish(uuid.NewString(), loaded, testdata.Now().Add(time.Minute))
		if err != nil {
			return err
		}
		if err := tx.InsertRevision(ctx, agentconfigport.StoredRevision{HR: &published}); err != nil {
			return err
		}
		if err := tx.SaveInstance(ctx, current); err != nil {
			return err
		}
		return tx.InsertCommand(ctx, agentconfigport.CommandIdentity{Scope: account.ID, Kind: acpCommandKind, Key: uuid.NewString()}, agentconfigport.CommandResult{
			Fingerprint: testdata.TokenHash, InstanceID: id, RevisionID: published.Snapshot().Identity.ID,
		})
	})
	require.Error(t, err)
	loaded, err := store.FindInstance(ctx, id)
	require.NoError(t, err)
	require.Nil(t, loaded.HR.Snapshot().Active)
	var revisions int64
	require.NoError(t, db.Raw("SELECT COUNT(*) FROM agent_revisions WHERE instance_id = ?", id).Scan(&revisions).Error)
	require.Zero(t, revisions)
	var commands int64
	require.NoError(t, db.Raw("SELECT COUNT(*) FROM agent_publication_commands WHERE scope = ?", account.ID).Scan(&commands).Error)
	require.Zero(t, commands)
}

// proveReplayAfterLaterPublication retains the original command result after a newer active revision.
func proveReplayAfterLaterPublication(t *testing.T, store agentconfigrepository.Store, db *gorm.DB) {
	ctx := context.Background()
	account := insertACPAccount(t, db)
	instance := persistHR(t, store, uuid.NewString(), mustACPValidHR(t), 1)
	id := instance.HR.Snapshot().ID
	firstVal := insertOwnedValidation(t, store, id, 1, true)
	firstRev := publishOwned(t, store, id, firstVal, account.ID, acpPublishKey, testdata.TokenHash)
	require.NoError(t, store.WithinInstance(ctx, id, func(tx agentconfigport.InstanceTx) error {
		current, err := tx.Instance(ctx)
		if err != nil {
			return err
		}
		expected, err := aggregate.NewDraftVersion(1)
		if err != nil {
			return err
		}
		if err := current.HR.ReplaceDraft(expected, mustACPEmptyHR(t), testdata.Now().Add(time.Minute)); err != nil {
			return err
		}
		return tx.SaveInstance(ctx, current)
	}))
	secondVal := insertOwnedValidation(t, store, id, 2, true)
	secondRev := publishOwned(t, store, id, secondVal, account.ID, uuid.NewString(), "ccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc3")
	got, found, err := store.FindCommand(ctx, agentconfigport.CommandIdentity{Scope: account.ID, Kind: acpCommandKind, Key: acpPublishKey})
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, firstRev, got.RevisionID)
	require.NotEqual(t, secondRev, got.RevisionID)
	loaded, err := store.FindInstance(ctx, id)
	require.NoError(t, err)
	require.Equal(t, secondRev, loaded.HR.Snapshot().Active.ID)
	original, err := store.FindRevision(ctx, id, firstRev)
	require.NoError(t, err)
	require.Equal(t, uint64(1), original.HR.Snapshot().Identity.Number)
}

// proveCorruptRestoration classifies malformed stored JSON as a technical failure.
func proveCorruptRestoration(t *testing.T, store agentconfigrepository.Store, db *gorm.DB) {
	ctx := context.Background()
	instance := persistHR(t, store, uuid.NewString(), mustACPValidHR(t), 1)
	id := instance.HR.Snapshot().ID
	require.NoError(t, db.Exec("UPDATE agent_drafts SET configuration = 'null' WHERE instance_id = ?", id).Error)
	_, err := store.FindInstance(ctx, id)
	var technical errs.Technical
	require.ErrorAs(t, err, &technical)
}

// proveNullZeroAndRestart writes empty JSON and null active pointers and retains revisions across reconnect.
func proveNullZeroAndRestart(t *testing.T, store agentconfigrepository.Store, db *gorm.DB) {
	ctx := context.Background()
	account := insertACPAccount(t, db)
	id := uuid.NewString()
	require.NoError(t, store.Create(ctx, restoreACPInstance(t, id, 1, mustACPEmptyHR(t), nil)))
	loaded, err := store.FindInstance(ctx, id)
	require.NoError(t, err)
	require.Nil(t, loaded.HR.Snapshot().Active)
	require.Empty(t, loaded.HR.Snapshot().Configuration.MandatoryQuestions())
	val := insertOwnedValidation(t, store, id, 1, false)
	require.NotEmpty(t, val)
	var eligible bool
	require.NoError(t, db.Raw("SELECT eligible FROM agent_validations WHERE id = ?", val).Scan(&eligible).Error)
	require.False(t, eligible)
	valid := insertOwnedValidation(t, store, id, 1, true)
	rev := publishOwned(t, store, id, valid, account.ID, uuid.NewString(), testdata.TokenHash)
	url := os.Getenv("TEST_DATABASE_URL")
	reopened, err := gorm.Open(postgres.Open(url), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	require.NoError(t, err)
	pool, err := reopened.DB()
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, pool.Close()) })
	again := agentconfigrepository.Store{DB: reopened}
	got, err := again.FindRevision(ctx, id, rev)
	require.NoError(t, err)
	require.Equal(t, rev, got.HR.Snapshot().Identity.ID)
}

// insertACPAccount stores a unique administrative account for publication scope.
func insertACPAccount(t *testing.T, db *gorm.DB) accessport.AccountRecord {
	t.Helper()
	store := accessrepository.Store{DB: db}
	record := testdata.AdministrativeAccountRecord(t)
	record.ID = uuid.NewString()
	record.Email = "acp-" + uuid.NewString() + "@example.com"
	record.Role = enum.HR
	require.NoError(t, store.WithinAccounts(context.Background(), func(tx accessport.AccountTx) error {
		return tx.Insert(context.Background(), record)
	}))
	return record
}

// persistHR creates an unpublished HR instance through the repository contract.
func persistHR(t *testing.T, store agentconfigrepository.Store, id string, cfg valueobject.HRConfiguration, version uint64) agentconfigport.StoredInstance {
	t.Helper()
	stored := restoreACPInstance(t, id, version, cfg, nil)
	require.NoError(t, store.Create(context.Background(), stored))
	return stored
}

// persistDocument creates an unpublished Document instance through the repository contract.
func persistDocument(t *testing.T, store agentconfigrepository.Store, id string, fields []string) agentconfigport.StoredInstance {
	t.Helper()
	cfg, err := valueobject.NewDocumentConfiguration(fields)
	require.NoError(t, err)
	version, err := aggregate.NewDraftVersion(1)
	require.NoError(t, err)
	instance, err := aggregate.NewInstance(aggregate.InstanceSnapshot[valueobject.DocumentConfiguration]{
		ID: id, Template: aggregate.TemplateRef{Template: catalog.TemplateDocumentAnalyzer, Version: catalog.TemplateVersion1, Behavior: catalog.BehaviorVersion1},
		DraftVersion: version, Configuration: cfg, CreatedAt: testdata.Now(), UpdatedAt: testdata.Now(),
	})
	require.NoError(t, err)
	stored := agentconfigport.StoredInstance{Name: "Invoice reviewer", Document: instance}
	require.NoError(t, store.Create(context.Background(), stored))
	return stored
}

// insertOwnedValidation records an assessment against the locked current draft.
func insertOwnedValidation(t *testing.T, store agentconfigrepository.Store, instanceID string, draft uint64, eligible bool) string {
	t.Helper()
	id := uuid.NewString()
	require.NoError(t, store.WithinInstance(context.Background(), instanceID, func(tx agentconfigport.InstanceTx) error {
		current, err := tx.Instance(context.Background())
		if err != nil {
			return err
		}
		template, err := storedTemplate(current)
		if err != nil {
			return err
		}
		return tx.InsertValidation(context.Background(), mustACPValidation(t, id, instanceID, draft, eligible, template))
	}))
	return id
}

// publishOwned commits a revision, active pointer and command for the locked instance.
func publishOwned(t *testing.T, store agentconfigrepository.Store, instanceID, validationID, scope, key, fingerprint string) string {
	t.Helper()
	var revisionID string
	require.NoError(t, store.WithinInstance(context.Background(), instanceID, func(tx agentconfigport.InstanceTx) error {
		current, err := tx.Instance(context.Background())
		if err != nil {
			return err
		}
		loaded, err := tx.FindValidation(context.Background(), validationID)
		if err != nil {
			return err
		}
		var stored agentconfigport.StoredRevision
		now := testdata.Now().Add(time.Minute)
		switch {
		case current.HR != nil:
			published, err := current.HR.Publish(uuid.NewString(), loaded, now)
			if err != nil {
				return err
			}
			stored = agentconfigport.StoredRevision{HR: &published}
			revisionID = published.Snapshot().Identity.ID
		case current.Document != nil:
			published, err := current.Document.Publish(uuid.NewString(), loaded, now)
			if err != nil {
				return err
			}
			stored = agentconfigport.StoredRevision{Document: &published}
			revisionID = published.Snapshot().Identity.ID
		default:
			return errors.New("missing instance template")
		}
		if err := tx.InsertRevision(context.Background(), stored); err != nil {
			return err
		}
		if err := tx.SaveInstance(context.Background(), current); err != nil {
			return err
		}
		return tx.InsertCommand(context.Background(), agentconfigport.CommandIdentity{Scope: scope, Kind: acpCommandKind, Key: key}, agentconfigport.CommandResult{
			Fingerprint: fingerprint, InstanceID: instanceID, RevisionID: revisionID,
		})
	}))
	return revisionID
}

// restoreACPInstance restores an unpublished HR fixture.
func restoreACPInstance(t *testing.T, id string, version uint64, cfg valueobject.HRConfiguration, active *aggregate.RevisionIdentity) agentconfigport.StoredInstance {
	t.Helper()
	draft, err := aggregate.NewDraftVersion(version)
	require.NoError(t, err)
	instance, err := aggregate.RestoreInstance(aggregate.InstanceSnapshot[valueobject.HRConfiguration]{
		ID: id, Template: aggregate.TemplateRef{Template: catalog.TemplateHRInterviewer, Version: catalog.TemplateVersion1, Behavior: catalog.BehaviorVersion1},
		DraftVersion: draft, Configuration: cfg, Active: active, CreatedAt: testdata.Now(), UpdatedAt: testdata.Now(),
	})
	require.NoError(t, err)
	return agentconfigport.StoredInstance{Name: "Invoice reviewer", HR: instance}
}

// mustACPValidHR constructs the approved valid HR example.
func mustACPValidHR(t *testing.T) valueobject.HRConfiguration {
	t.Helper()
	cfg, err := valueobject.NewHRConfiguration(valueobject.HRInput{
		JobDescriptionRef: &valueobject.RefInput{ID: "11111111-1111-4111-8111-111111111111", Version: "1"},
		MandatoryQuestions: []valueobject.QuestionInput{
			{ID: "background", Stage: "background", Text: "Describe your backend experience."},
			{ID: "screening", Stage: "screening", Text: "How do you protect a concurrent update?"},
			{ID: "compensation", Stage: "compensation", Text: "What are your compensation expectations?"},
		},
		CompetencyTopics: []string{"concurrency"},
		StageSeconds:     &valueobject.StageSecondsInput{Background: 60, Screening: 240, Compensation: 90, Closing: 30},
		CompensationPolicy: &valueobject.CompensationInput{
			DiscussionMode:   "expectations_only",
			SalaryCeilingRef: &valueobject.RefInput{ID: "22222222-2222-4222-8222-222222222222", Version: "1"},
		},
	})
	require.NoError(t, err)
	return cfg
}

// mustACPEmptyHR constructs an incomplete but storage-valid HR draft.
func mustACPEmptyHR(t *testing.T) valueobject.HRConfiguration {
	t.Helper()
	cfg, err := valueobject.NewHRConfiguration(valueobject.HRInput{})
	require.NoError(t, err)
	return cfg
}

// mustACPValidation restores an exact-version assessment fixture.
func mustACPValidation(t *testing.T, id, instanceID string, draft uint64, eligible bool, template aggregate.TemplateRef) aggregate.Validation {
	t.Helper()
	version, err := aggregate.NewDraftVersion(draft)
	require.NoError(t, err)
	validation, err := aggregate.NewValidation(aggregate.ValidationSnapshot{
		ID: id, InstanceID: instanceID, DraftVersion: version, Template: template, Eligible: eligible, CreatedAt: testdata.Now(),
	})
	require.NoError(t, err)
	return validation
}

// storedTemplate returns the locked instance catalog identity.
func storedTemplate(stored agentconfigport.StoredInstance) (aggregate.TemplateRef, error) {
	switch {
	case stored.HR != nil:
		return stored.HR.Snapshot().Template, nil
	case stored.Document != nil:
		return stored.Document.Snapshot().Template, nil
	default:
		return aggregate.TemplateRef{}, errors.New("missing instance template")
	}
}
