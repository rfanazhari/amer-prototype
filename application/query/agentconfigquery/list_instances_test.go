package agentconfigquery_test

import (
	"context"
	"encoding/json"
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

// listInstancesCase names one collection-read outcome for every stored instance.
type listInstancesCase struct {
	name         string
	principalErr error
	authorizeErr error
	stored       []agentconfigport.StoredInstance
	wantIDs      []string
	wantErr      bool
	err          error
	wantEmpty    bool
}

// storedHRAt restores one HR instance identified by id for collection-read fixtures.
func storedHRAt(t *testing.T, id, name string) agentconfigport.StoredInstance {
	t.Helper()
	instance, err := aggregate.RestoreInstance(aggregate.InstanceSnapshot[valueobject.HRConfiguration]{
		ID: id, Template: aggregate.TemplateRef{Template: catalog.TemplateHRInterviewer, Version: catalog.TemplateVersion1, Behavior: catalog.BehaviorVersion1},
		DraftVersion: mustVersion(t, 3), Configuration: mustHR(t, validHRInput()),
		CreatedAt: testdata.Now(), UpdatedAt: testdata.Now(),
	})
	require.NoError(t, err)
	return agentconfigport.StoredInstance{Name: name, HR: instance}
}

// TestListInstancesReturnsEveryFilteredInstance lists every restored instance and strips salary contents.
func TestListInstancesReturnsEveryFilteredInstance(t *testing.T) {
	secondID := "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaa9"
	tests := []listInstancesCase{
		{
			name: "every restored instance",
			stored: []agentconfigport.StoredInstance{
				storedHRAt(t, instanceID, instanceName),
				storedHRAt(t, secondID, "Payroll reviewer"),
			},
			wantIDs: []string{instanceID, secondID},
		},
		{name: "authorization failure performs no read", authorizeErr: service.ErrForbidden, wantErr: true, err: service.ErrForbidden},
		{name: "missing principal performs no read", principalErr: service.ErrCandidateSessionUnbound, wantErr: true, err: service.ErrCandidateSessionUnbound},
		{name: "empty list", stored: []agentconfigport.StoredInstance{}, wantEmpty: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			authority := mocks.NewAgentConfigurationAuthority(t)
			lister := mocks.NewInstanceLister(t)
			if tt.principalErr != nil {
				authority.On("Principal", mock.Anything).Return(service.AdminPrincipal{}, tt.principalErr).Once()
			} else {
				authority.On("Principal", mock.Anything).Return(hrPrincipal(), nil).Once()
				authority.On("Authorize", mock.Anything, hrPrincipal(), service.AgentConfigurationRead, "").Return(tt.authorizeErr).Once()
			}
			if !tt.wantErr {
				lister.On("ListInstances", mock.Anything).Return(tt.stored, nil).Once()
			}
			got, err := (agentconfigquery.ListInstances{Authority: authority, Lister: lister}).Execute(context.Background())
			if tt.wantErr {
				require.ErrorIs(t, err, tt.err)
				assert.Nil(t, got)
				lister.AssertNotCalled(t, "ListInstances", mock.Anything)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, got)
			if tt.wantEmpty {
				assert.Empty(t, got)
				return
			}
			require.Len(t, got, len(tt.wantIDs))
			for i, id := range tt.wantIDs {
				assert.Equal(t, id, got[i].ID)
				require.NotNil(t, got[i].Configuration.HR)
				require.NotNil(t, got[i].Configuration.HR.CompensationPolicy)
				assert.True(t, got[i].Configuration.HR.CompensationPolicy.SalaryCeilingConfigured)
				body, marshalErr := json.Marshal(got[i].Configuration)
				require.NoError(t, marshalErr)
				assert.NotContains(t, string(body), salaryCeilingID)
			}
		})
	}
}
