package agentconfigusecase_test

import (
	"context"
	"testing"

	"github.com/rfanazhari/amer-prototype/application/repository/agentconfigport"
	"github.com/rfanazhari/amer-prototype/application/service"
	"github.com/rfanazhari/amer-prototype/application/usecase/agentconfigusecase"
	catalog "github.com/rfanazhari/amer-prototype/domain/agentconfig"
	aggregate "github.com/rfanazhari/amer-prototype/domain/aggregate/agentconfig"
	"github.com/rfanazhari/amer-prototype/test/mocks"
	"github.com/rfanazhari/amer-prototype/test/testdata"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// createCase names one create-command authorization and persistence outcome.
type createCase struct {
	name         string
	principalErr error
	authorizeErr error
	input        agentconfigusecase.CreateInput
	wantErr      error
	wantCreate   bool
	document     bool
}

// TestCreateAuthorizesHRAndPersistsDetachedDrafts refuses ADMIN and candidate actors before storage.
func TestCreateAuthorizesHRAndPersistsDetachedDrafts(t *testing.T) {
	hr := validHRInput()
	tests := []createCase{
		{name: "hr allow hr template", input: agentconfigusecase.CreateInput{TemplateID: catalog.TemplateHRInterviewer, TemplateVersion: catalog.TemplateVersion1, Name: "  Invoice reviewer  ", HR: &hr}, wantCreate: true},
		{name: "hr allow document template", input: agentconfigusecase.CreateInput{TemplateID: catalog.TemplateDocumentAnalyzer, TemplateVersion: catalog.TemplateVersion1, Name: instanceName, DocumentFields: []string{"total"}}, wantCreate: true, document: true},
		{name: "admin deny", authorizeErr: service.ErrForbidden, input: agentconfigusecase.CreateInput{TemplateID: catalog.TemplateHRInterviewer, TemplateVersion: catalog.TemplateVersion1, Name: instanceName, HR: &hr}, wantErr: service.ErrForbidden},
		{name: "candidate deny", principalErr: service.ErrCandidateSessionUnbound, input: agentconfigusecase.CreateInput{TemplateID: catalog.TemplateHRInterviewer, TemplateVersion: catalog.TemplateVersion1, Name: instanceName, HR: &hr}, wantErr: service.ErrCandidateSessionUnbound},
		{name: "unknown template", input: agentconfigusecase.CreateInput{TemplateID: "custom-agent", TemplateVersion: catalog.TemplateVersion1, Name: instanceName}, wantErr: catalog.ErrUnknownTemplate},
		{name: "invalid name", input: agentconfigusecase.CreateInput{TemplateID: catalog.TemplateHRInterviewer, TemplateVersion: catalog.TemplateVersion1, Name: "   "}, wantErr: service.ErrInvalidInput},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			authority := mocks.NewAgentConfigurationAuthority(t)
			instances := mocks.NewMutation[agentconfigport.StoredInstance, aggregate.Validation, agentconfigport.StoredRevision](t)
			clock := mocks.NewClock(t)
			if tt.principalErr != nil {
				authority.On("Principal", mock.Anything).Return(service.AdminPrincipal{}, tt.principalErr).Once()
			} else {
				authority.On("Principal", mock.Anything).Return(hrPrincipal(), nil).Once()
				authority.On("Authorize", mock.Anything, hrPrincipal(), service.AgentConfigurationCreate, "").Return(tt.authorizeErr).Once()
			}
			if tt.wantCreate {
				clock.On("Now").Return(testdata.Now())
				instances.On("Create", mock.Anything, mock.Anything).Return(func(_ context.Context, stored agentconfigport.StoredInstance) error {
					require.Equal(t, instanceName, stored.Name)
					if tt.document {
						require.NotNil(t, stored.Document)
						require.Nil(t, stored.HR)
						snap := stored.Document.Snapshot()
						require.Equal(t, uint64(1), snap.DraftVersion.Number())
						require.Nil(t, snap.Active)
						require.Equal(t, testdata.Now(), snap.CreatedAt)
						return nil
					}
					require.NotNil(t, stored.HR)
					require.Nil(t, stored.Document)
					snap := stored.HR.Snapshot()
					require.Equal(t, uint64(1), snap.DraftVersion.Number())
					require.Nil(t, snap.Active)
					require.NotNil(t, snap.Configuration.CompensationPolicy())
					require.NotNil(t, snap.Configuration.CompensationPolicy().SalaryCeilingRef())
					return nil
				}).Once()
			}
			got, err := (agentconfigusecase.Create{Authority: authority, Instances: instances, Clock: clock}).Execute(context.Background(), tt.input)
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				assert.Nil(t, got)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, got)
			assert.Equal(t, instanceName, got.Name)
			assert.Equal(t, uint64(1), got.DraftVersion)
			assert.Nil(t, got.ActiveRevisionID)
			assert.Equal(t, testdata.Now(), got.CreatedAt)
			assert.Equal(t, testdata.Now(), got.UpdatedAt)
			if tt.document {
				assert.Equal(t, catalog.TemplateDocumentAnalyzer, got.TemplateID)
				require.NotNil(t, got.Configuration.Document)
				assert.Equal(t, []string{"total"}, got.Configuration.Document.RequiredFields)
				assert.Nil(t, got.Configuration.HR)
				return
			}
			assert.Equal(t, catalog.TemplateHRInterviewer, got.TemplateID)
			require.NotNil(t, got.Configuration.HR)
			require.NotNil(t, got.Configuration.HR.CompensationPolicy)
			assert.Equal(t, "expectations_only", got.Configuration.HR.CompensationPolicy.DiscussionMode)
			assert.True(t, got.Configuration.HR.CompensationPolicy.SalaryCeilingConfigured)
			assert.NotContains(t, got.Configuration.HR.CompensationPolicy.DiscussionMode, salaryCeilingID)
			assert.Nil(t, got.Configuration.Document)
		})
	}
}
