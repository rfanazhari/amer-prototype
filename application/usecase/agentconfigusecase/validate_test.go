package agentconfigusecase_test

import (
	"context"
	"errors"
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

// validateCase names one exact-version validation recording outcome.
type validateCase struct {
	name       string
	locked     uint64
	refErr     error
	wantErr    error
	wantInsert bool
}

// TestValidateRecordsExactVersionAndRejectsEditRaces resolves references before the lock.
func TestValidateRecordsExactVersionAndRejectsEditRaces(t *testing.T) {
	unavailable := errors.New("reference store unavailable")
	tests := []validateCase{
		{name: "hr allow eligible", locked: 3, wantInsert: true},
		{name: "edit race rejects insert", locked: 4, wantErr: service.ErrDraftConflict},
		{name: "reference failure records no assessment", refErr: unavailable, wantErr: unavailable},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			authority := mocks.NewAgentConfigurationAuthority(t)
			reader := mocks.NewReader[agentconfigport.StoredInstance, agentconfigport.StoredRevision](t)
			instances := mocks.NewMutation[agentconfigport.StoredInstance, aggregate.Validation, agentconfigport.StoredRevision](t)
			clock := mocks.NewClock(t)
			references := mocks.NewAgentReferenceReader[service.AgentReferenceFact](t)
			authority.On("Principal", mock.Anything).Return(hrPrincipal(), nil).Once()
			authority.On("Authorize", mock.Anything, hrPrincipal(), service.AgentConfigurationValidate, instanceID).Return(nil).Once()
			reader.On("FindInstance", mock.Anything, instanceID).Return(storedHR(t, 3, mustHR(t, validHRInput()), nil), nil).Once()
			references.On("Resolve", mock.Anything, service.AgentReference{Kind: service.ReferenceKindJobDescription, ID: jobDescriptionID, Version: "1"}).
				Return(service.AgentReferenceFact{Kind: service.ReferenceKindJobDescription, ID: jobDescriptionID, Version: "1", Role: service.DemoJobRole, Ready: true}, true, tt.refErr).Once()
			if tt.refErr == nil {
				references.On("Resolve", mock.Anything, service.AgentReference{Kind: service.ReferenceKindCompensationPolicy, ID: salaryCeilingID, Version: "1"}).
					Return(service.AgentReferenceFact{Kind: service.ReferenceKindCompensationPolicy, ID: salaryCeilingID, Version: "1", Ready: true}, true, nil).Once()
			}
			if tt.refErr == nil {
				instances.On("WithinInstance", mock.Anything, instanceID, mock.Anything).Return(func(ctx context.Context, _ string, fn func(agentconfigport.InstanceTx) error) error {
					tx := mocks.NewTransaction[agentconfigport.StoredInstance, aggregate.Validation, agentconfigport.StoredRevision](t)
					tx.On("Instance", mock.Anything).Return(storedHR(t, tt.locked, mustHR(t, validHRInput()), nil), nil).Once()
					if tt.wantInsert {
						tx.On("InsertValidation", mock.Anything, mock.Anything).Return(func(_ context.Context, validation aggregate.Validation) error {
							snap := validation.Snapshot()
							require.Equal(t, instanceID, snap.InstanceID)
							require.Equal(t, uint64(3), snap.DraftVersion.Number())
							require.True(t, snap.Eligible)
							require.Empty(t, snap.Findings)
							return nil
						}).Once()
					}
					return fn(tx)
				}).Once()
			}
			if tt.wantInsert {
				clock.On("Now").Return(testdata.Now())
			}
			got, err := (agentconfigusecase.Validate{
				Authority:  authority,
				Reader:     reader,
				Instances:  instances,
				Assessment: service.DraftAssessment{References: references},
				Clock:      clock,
			}).Execute(context.Background(), agentconfigusecase.ValidateInput{InstanceID: instanceID, ExpectedDraftVersion: 3})
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				assert.Nil(t, got)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, got)
			assert.Equal(t, instanceID, got.InstanceID)
			assert.Equal(t, uint64(3), got.DraftVersion)
			assert.Equal(t, catalog.TemplateHRInterviewer, got.TemplateID)
			assert.True(t, got.Eligible)
			assert.Empty(t, got.Findings)
			assert.Equal(t, testdata.Now(), got.CreatedAt)
		})
	}
}
