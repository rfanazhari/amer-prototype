package agentconfigrepository_test

import (
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/rfanazhari/amer-prototype/application/repository/agentconfigport"
	catalog "github.com/rfanazhari/amer-prototype/domain/agentconfig"
	aggregate "github.com/rfanazhari/amer-prototype/domain/aggregate/agentconfig"
	"github.com/rfanazhari/amer-prototype/domain/valueobject"
	"github.com/rfanazhari/amer-prototype/infrastructure/repository/agentconfigrepository"
	"github.com/rfanazhari/amer-prototype/test/testdata"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

const (
	instanceID       = "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaa8"
	otherInstanceID  = "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaa9"
	validationID     = "bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbb8"
	revisionID       = "cccccccc-cccc-4ccc-8ccc-ccccccccccc8"
	jobDescriptionID = "11111111-1111-4111-8111-111111111111"
	salaryCeilingID  = "22222222-2222-4222-8222-222222222222"
	publishKey       = "publish-invoice-007"
	commandKind      = "publish-agent-configuration/v1"
	instanceName     = "Invoice reviewer"
	emptyHRJSON      = `{"jobDescriptionRef":null,"mandatoryQuestions":[],"competencyTopics":[],"stageSeconds":null,"compensationPolicy":null}`
	emptyFindings    = `[]`
	listedSalaryJSON = `{"jobDescriptionRef":{"id":"11111111-1111-4111-8111-111111111111","version":"1"},"mandatoryQuestions":[{"id":"background","stage":"background","text":"Describe your backend experience."},{"id":"screening","stage":"screening","text":"How do you protect a concurrent update?"},{"id":"compensation","stage":"compensation","text":"What are your compensation expectations?"}],"competencyTopics":["concurrency"],"stageSeconds":{"background":60,"screening":240,"compensation":90,"closing":30},"compensationPolicy":{"discussionMode":"expectations_only","salaryCeilingRef":{"id":"22222222-2222-4222-8222-222222222222","version":"1"}}}`
)

// sqlStore opens a GORM adapter over sqlmock with ping disabled so tests own SQL expectations.
func sqlStore(t *testing.T) (agentconfigrepository.Store, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, mock.ExpectationsWereMet())
		_ = db.Close()
	})
	gdb, err := gorm.Open(postgres.New(postgres.Config{Conn: db, DriverName: "postgres"}), &gorm.Config{
		Logger:                 logger.Default.LogMode(logger.Silent),
		SkipDefaultTransaction: true,
		DisableAutomaticPing:   true,
	})
	require.NoError(t, err)
	return agentconfigrepository.Store{DB: gdb}, mock
}

// validHRInput returns the approved valid HR configuration example.
func validHRInput() valueobject.HRInput {
	return valueobject.HRInput{
		JobDescriptionRef: &valueobject.RefInput{ID: jobDescriptionID, Version: "1"},
		MandatoryQuestions: []valueobject.QuestionInput{
			{ID: "background", Stage: "background", Text: "Describe your backend experience."},
			{ID: "screening", Stage: "screening", Text: "How do you protect a concurrent update?"},
			{ID: "compensation", Stage: "compensation", Text: "What are your compensation expectations?"},
		},
		CompetencyTopics: []string{"concurrency"},
		StageSeconds:     &valueobject.StageSecondsInput{Background: 60, Screening: 240, Compensation: 90, Closing: 30},
		CompensationPolicy: &valueobject.CompensationInput{
			DiscussionMode:   "expectations_only",
			SalaryCeilingRef: &valueobject.RefInput{ID: salaryCeilingID, Version: "1"},
		},
	}
}

// mustHR constructs a storage-valid HR draft or fails the test.
func mustHR(t *testing.T, in valueobject.HRInput) valueobject.HRConfiguration {
	t.Helper()
	cfg, err := valueobject.NewHRConfiguration(in)
	require.NoError(t, err)
	return cfg
}

// hrTemplate is the catalog HR Interviewer v1 reference.
func hrTemplate() aggregate.TemplateRef {
	return aggregate.TemplateRef{Template: catalog.TemplateHRInterviewer, Version: catalog.TemplateVersion1, Behavior: catalog.BehaviorVersion1}
}

// mustVersion constructs a positive draft version or fails the test.
func mustVersion(t *testing.T, n uint64) aggregate.DraftVersion {
	t.Helper()
	v, err := aggregate.NewDraftVersion(n)
	require.NoError(t, err)
	return v
}

// storedHR restores an HR instance snapshot for persistence tests.
func storedHR(t *testing.T, id string, version uint64, cfg valueobject.HRConfiguration, active *aggregate.RevisionIdentity) agentconfigport.StoredInstance {
	t.Helper()
	instance, err := aggregate.RestoreInstance(aggregate.InstanceSnapshot[valueobject.HRConfiguration]{
		ID: id, Template: hrTemplate(), DraftVersion: mustVersion(t, version), Configuration: cfg,
		Active: active, CreatedAt: testdata.Now(), UpdatedAt: testdata.Now(),
	})
	require.NoError(t, err)
	return agentconfigport.StoredInstance{Name: instanceName, HR: instance}
}

// storedHRRevision restores an immutable HR revision.
func storedHRRevision(t *testing.T, id string, number, draft uint64, cfg valueobject.HRConfiguration) agentconfigport.StoredRevision {
	t.Helper()
	revision, err := aggregate.RestoreRevision(aggregate.RevisionSnapshot[valueobject.HRConfiguration]{
		Identity: aggregate.RevisionIdentity{
			ID: id, InstanceID: instanceID, Number: number, DraftVersion: mustVersion(t, draft), Template: hrTemplate(),
		},
		ValidationID: validationID, Configuration: cfg, CreatedAt: testdata.Now(),
	})
	require.NoError(t, err)
	return agentconfigport.StoredRevision{HR: &revision}
}

// commandIdentity is the subject-scoped publication identity for the fixture account.
func commandIdentity() agentconfigport.CommandIdentity {
	return agentconfigport.CommandIdentity{Scope: testdata.ID, Kind: commandKind, Key: publishKey}
}

// eligibleValidation restores an exact-version eligible assessment.
func eligibleValidation(t *testing.T, id, instance string, draft uint64, eligible bool) aggregate.Validation {
	t.Helper()
	validation, err := aggregate.NewValidation(aggregate.ValidationSnapshot{
		ID: id, InstanceID: instance, DraftVersion: mustVersion(t, draft),
		Template: hrTemplate(), Eligible: eligible, Findings: nil, CreatedAt: testdata.Now(),
	})
	require.NoError(t, err)
	return validation
}

// fingerprint is a independently specified 64-character lowercase SHA-256 hex digest.
func fingerprint() string { return testdata.TokenHash }
