package agentconfigusecase_test

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strings"
	"testing"

	"github.com/rfanazhari/amer-prototype/application/repository/agentconfigport"
	"github.com/rfanazhari/amer-prototype/application/service"
	catalog "github.com/rfanazhari/amer-prototype/domain/agentconfig"
	aggregate "github.com/rfanazhari/amer-prototype/domain/aggregate/agentconfig"
	"github.com/rfanazhari/amer-prototype/domain/valueobject"
	"github.com/rfanazhari/amer-prototype/test/testdata"
	"github.com/stretchr/testify/require"
)

const (
	instanceID       = "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaa8"
	validationID     = "bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbb8"
	revisionID       = "cccccccc-cccc-4ccc-8ccc-ccccccccccc8"
	laterRevisionID  = "dddddddd-dddd-4ddd-8ddd-ddddddddddd8"
	jobDescriptionID = "11111111-1111-4111-8111-111111111111"
	salaryCeilingID  = "22222222-2222-4222-8222-222222222222"
	publishKey       = "publish-invoice-007"
	commandKind      = "publish-agent-configuration/v1"
	instanceName     = "Invoice reviewer"
)

// hrPrincipal is the trusted HR account UUID used as publication scope.
func hrPrincipal() service.AdminPrincipal {
	return service.AdminPrincipal{SubjectID: testdata.ID}
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

// mustDocument constructs a storage-valid Document draft or fails the test.
func mustDocument(t *testing.T, fields []string) valueobject.DocumentConfiguration {
	t.Helper()
	cfg, err := valueobject.NewDocumentConfiguration(fields)
	require.NoError(t, err)
	return cfg
}

// hrTemplate is the catalog HR Interviewer v1 reference.
func hrTemplate() aggregate.TemplateRef {
	return aggregate.TemplateRef{Template: catalog.TemplateHRInterviewer, Version: catalog.TemplateVersion1, Behavior: catalog.BehaviorVersion1}
}

// documentTemplate is the catalog Document Analyzer v1 reference.
func documentTemplate() aggregate.TemplateRef {
	return aggregate.TemplateRef{Template: catalog.TemplateDocumentAnalyzer, Version: catalog.TemplateVersion1, Behavior: catalog.BehaviorVersion1}
}

// mustVersion constructs a positive draft version or fails the test.
func mustVersion(t *testing.T, n uint64) aggregate.DraftVersion {
	t.Helper()
	v, err := aggregate.NewDraftVersion(n)
	require.NoError(t, err)
	return v
}

// storedHR restores an HR instance snapshot for repository mock returns.
func storedHR(t *testing.T, version uint64, cfg valueobject.HRConfiguration, active *aggregate.RevisionIdentity) agentconfigport.StoredInstance {
	t.Helper()
	instance, err := aggregate.RestoreInstance(aggregate.InstanceSnapshot[valueobject.HRConfiguration]{
		ID: instanceID, Template: hrTemplate(), DraftVersion: mustVersion(t, version), Configuration: cfg,
		Active: active, CreatedAt: testdata.Now(), UpdatedAt: testdata.Now(),
	})
	require.NoError(t, err)
	return agentconfigport.StoredInstance{Name: instanceName, HR: instance}
}

// storedDocument restores a Document instance snapshot for repository mock returns.
func storedDocument(t *testing.T, version uint64, cfg valueobject.DocumentConfiguration) agentconfigport.StoredInstance {
	t.Helper()
	instance, err := aggregate.RestoreInstance(aggregate.InstanceSnapshot[valueobject.DocumentConfiguration]{
		ID: instanceID, Template: documentTemplate(), DraftVersion: mustVersion(t, version), Configuration: cfg,
		CreatedAt: testdata.Now(), UpdatedAt: testdata.Now(),
	})
	require.NoError(t, err)
	return agentconfigport.StoredInstance{Name: instanceName, Document: instance}
}

// storedHRRevision restores an immutable HR revision for replay reads.
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

// expectedFingerprint independently computes the approved publication intent digest.
func expectedFingerprint(instanceUUID string, draftVersion uint64, validationUUID string) string {
	payload, err := json.Marshal([]any{commandKind, strings.ToLower(instanceUUID), draftVersion, strings.ToLower(validationUUID)})
	if err != nil {
		panic(err)
	}
	sum := sha256.Sum256(payload)
	return hex.EncodeToString(sum[:])
}

// commandIdentity is the subject-scoped publication identity for the fixture instance.
func commandIdentity() agentconfigport.CommandIdentity {
	return agentconfigport.CommandIdentity{Scope: testdata.ID, Kind: commandKind, Key: publishKey}
}

// eligibleValidation restores an exact-version eligible assessment for the fixture instance.
func eligibleValidation(t *testing.T, draft uint64) aggregate.Validation {
	t.Helper()
	validation, err := aggregate.NewValidation(aggregate.ValidationSnapshot{
		ID: validationID, InstanceID: instanceID, DraftVersion: mustVersion(t, draft),
		Template: hrTemplate(), Eligible: true, CreatedAt: testdata.Now(),
	})
	require.NoError(t, err)
	return validation
}
