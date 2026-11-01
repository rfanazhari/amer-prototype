package agentconfigquery_test

import (
	"context"
	"testing"

	"github.com/rfanazhari/amer-prototype/application/query/agentconfigquery"
	"github.com/rfanazhari/amer-prototype/application/repository/agentconfigport"
	"github.com/rfanazhari/amer-prototype/application/service"
	catalog "github.com/rfanazhari/amer-prototype/domain/agentconfig"
	aggregate "github.com/rfanazhari/amer-prototype/domain/aggregate/agentconfig"
	"github.com/rfanazhari/amer-prototype/domain/valueobject"
	"github.com/rfanazhari/amer-prototype/test/mocks"
	"github.com/rfanazhari/amer-prototype/test/testdata"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

const (
	instanceID       = "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaa8"
	revisionID       = "cccccccc-cccc-4ccc-8ccc-ccccccccccc8"
	validationID     = "bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbb8"
	jobDescriptionID = "11111111-1111-4111-8111-111111111111"
	salaryCeilingID  = "22222222-2222-4222-8222-222222222222"
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

// mustVersion constructs a positive draft version or fails the test.
func mustVersion(t *testing.T, n uint64) aggregate.DraftVersion {
	t.Helper()
	v, err := aggregate.NewDraftVersion(n)
	require.NoError(t, err)
	return v
}

// storedHR restores an HR instance snapshot for repository mock returns.
func storedHR(t *testing.T) agentconfigport.StoredInstance {
	t.Helper()
	instance, err := aggregate.RestoreInstance(aggregate.InstanceSnapshot[valueobject.HRConfiguration]{
		ID: instanceID, Template: aggregate.TemplateRef{Template: catalog.TemplateHRInterviewer, Version: catalog.TemplateVersion1, Behavior: catalog.BehaviorVersion1},
		DraftVersion: mustVersion(t, 3), Configuration: mustHR(t, validHRInput()),
		CreatedAt: testdata.Now(), UpdatedAt: testdata.Now(),
	})
	require.NoError(t, err)
	return agentconfigport.StoredInstance{Name: instanceName, HR: instance}
}

// storedHRRevision restores an immutable HR revision for read queries.
func storedHRRevision(t *testing.T) agentconfigport.StoredRevision {
	t.Helper()
	revision, err := aggregate.RestoreRevision(aggregate.RevisionSnapshot[valueobject.HRConfiguration]{
		Identity: aggregate.RevisionIdentity{
			ID: revisionID, InstanceID: instanceID, Number: 1, DraftVersion: mustVersion(t, 3),
			Template: aggregate.TemplateRef{Template: catalog.TemplateHRInterviewer, Version: catalog.TemplateVersion1, Behavior: catalog.BehaviorVersion1},
		},
		ValidationID: validationID, Configuration: mustHR(t, validHRInput()), CreatedAt: testdata.Now(),
	})
	require.NoError(t, err)
	return agentconfigport.StoredRevision{HR: &revision}
}

// catalogCase names one catalog collection-read outcome.
type catalogCase struct {
	name         string
	principalErr error
	authorizeErr error
	wantErr      error
}

// TestCatalogAuthorizesCollectionReadListsBothTemplates uses the generated authority mock.
func TestCatalogAuthorizesCollectionReadListsBothTemplates(t *testing.T) {
	tests := []catalogCase{
		{name: "hr allow catalog"},
		{name: "admin deny", authorizeErr: service.ErrForbidden, wantErr: service.ErrForbidden},
		{name: "candidate deny", principalErr: service.ErrCandidateSessionUnbound, wantErr: service.ErrCandidateSessionUnbound},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			authority := mocks.NewAgentConfigurationAuthority(t)
			if tt.principalErr != nil {
				authority.On("Principal", mock.Anything).Return(service.AdminPrincipal{}, tt.principalErr).Once()
			} else {
				authority.On("Principal", mock.Anything).Return(hrPrincipal(), nil).Once()
				authority.On("Authorize", mock.Anything, hrPrincipal(), service.AgentConfigurationRead, "").Return(tt.authorizeErr).Once()
			}
			got, err := (agentconfigquery.Catalog{Authority: authority}).Execute(context.Background())
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				assert.Nil(t, got)
				return
			}
			require.NoError(t, err)
			require.Len(t, got, 2)
			assert.Equal(t, catalog.TemplateHRInterviewer, got[0].TemplateID)
			assert.Equal(t, catalog.TemplateDocumentAnalyzer, got[1].TemplateID)
			assert.Equal(t, catalog.WarningPolicyInformational, got[0].WarningPolicy)
		})
	}
}

// instanceReadCase names one instance-read authorization outcome.
type instanceReadCase struct {
	name         string
	principalErr error
	authorizeErr error
	wantErr      error
}

// TestInstanceReadAuthorizesBeforeDisclosure hides stored drafts from ADMIN and candidate actors.
func TestInstanceReadAuthorizesBeforeDisclosure(t *testing.T) {
	tests := []instanceReadCase{
		{name: "hr allow read"},
		{name: "admin deny", authorizeErr: service.ErrForbidden, wantErr: service.ErrForbidden},
		{name: "candidate deny", principalErr: service.ErrCandidateSessionUnbound, wantErr: service.ErrCandidateSessionUnbound},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			authority := mocks.NewAgentConfigurationAuthority(t)
			reader := mocks.NewReader[agentconfigport.StoredInstance, agentconfigport.StoredRevision](t)
			if tt.principalErr != nil {
				authority.On("Principal", mock.Anything).Return(service.AdminPrincipal{}, tt.principalErr).Once()
			} else {
				authority.On("Principal", mock.Anything).Return(hrPrincipal(), nil).Once()
				authority.On("Authorize", mock.Anything, hrPrincipal(), service.AgentConfigurationRead, instanceID).Return(tt.authorizeErr).Once()
			}
			if tt.wantErr == nil {
				reader.On("FindInstance", mock.Anything, instanceID).Return(storedHR(t), nil).Once()
			}
			got, err := (agentconfigquery.Instance{Authority: authority, Reader: reader}).Execute(context.Background(), instanceID)
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				assert.Nil(t, got)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, got)
			assert.Equal(t, instanceName, got.Name)
			assert.Equal(t, uint64(3), got.DraftVersion)
			require.NotNil(t, got.Configuration.HR.CompensationPolicy)
			assert.True(t, got.Configuration.HR.CompensationPolicy.SalaryCeilingConfigured)
		})
	}
}

// revisionReadCase names one revision-read outcome.
type revisionReadCase struct {
	name    string
	wantErr bool
}

// TestRevisionReadReturnsFilteredImmutableSnapshot uses the generated reader mock.
func TestRevisionReadReturnsFilteredImmutableSnapshot(t *testing.T) {
	tests := []revisionReadCase{{name: "hr allow revision", wantErr: false}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			authority := mocks.NewAgentConfigurationAuthority(t)
			reader := mocks.NewReader[agentconfigport.StoredInstance, agentconfigport.StoredRevision](t)
			authority.On("Principal", mock.Anything).Return(hrPrincipal(), nil).Once()
			authority.On("Authorize", mock.Anything, hrPrincipal(), service.AgentConfigurationRead, instanceID).Return(nil).Once()
			reader.On("FindRevision", mock.Anything, instanceID, revisionID).Return(storedHRRevision(t), nil).Once()
			got, err := (agentconfigquery.Revision{Authority: authority, Reader: reader}).Execute(context.Background(), instanceID, revisionID)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, got)
			assert.Equal(t, revisionID, got.RevisionID)
			assert.Equal(t, uint64(1), got.RevisionNumber)
			require.NotNil(t, got.Configuration.HR.CompensationPolicy)
			assert.True(t, got.Configuration.HR.CompensationPolicy.SalaryCeilingConfigured)
			assert.NotContains(t, got.Configuration.HR.CompensationPolicy.DiscussionMode, salaryCeilingID)
		})
	}
}
