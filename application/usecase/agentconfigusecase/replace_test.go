package agentconfigusecase_test

import (
	"context"
	"testing"

	"github.com/rfanazhari/amer-prototype/application/repository/agentconfigport"
	"github.com/rfanazhari/amer-prototype/application/service"
	"github.com/rfanazhari/amer-prototype/application/usecase/agentconfigusecase"
	aggregate "github.com/rfanazhari/amer-prototype/domain/aggregate/agentconfig"
	"github.com/rfanazhari/amer-prototype/test/mocks"
	"github.com/rfanazhari/amer-prototype/test/testdata"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// replaceCase names one exact-version replacement outcome.
type replaceCase struct {
	name     string
	expected uint64
	current  uint64
	wantErr  error
}

// TestReplaceRejectsStaleDraftsAndPreservesActiveSelection uses the generated mutation mock.
func TestReplaceRejectsStaleDraftsAndPreservesActiveSelection(t *testing.T) {
	hr := validHRInput()
	active := &aggregate.RevisionIdentity{ID: revisionID, InstanceID: instanceID, Number: 1, DraftVersion: mustVersion(t, 1), Template: hrTemplate()}
	tests := []replaceCase{
		{name: "hr allow replace", expected: 2, current: 2},
		{name: "stale expected version", expected: 1, current: 2, wantErr: service.ErrDraftConflict},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			authority := mocks.NewAgentConfigurationAuthority(t)
			instances := mocks.NewMutation[agentconfigport.StoredInstance, aggregate.Validation, agentconfigport.StoredRevision](t)
			clock := mocks.NewClock(t)
			authority.On("Principal", mock.Anything).Return(hrPrincipal(), nil).Once()
			authority.On("Authorize", mock.Anything, hrPrincipal(), service.AgentConfigurationReplace, instanceID).Return(nil).Once()
			clock.On("Now").Return(testdata.Now().AddDate(0, 0, 1))
			instances.On("WithinInstance", mock.Anything, instanceID, mock.Anything).Return(func(ctx context.Context, _ string, fn func(agentconfigport.InstanceTx) error) error {
				tx := mocks.NewTransaction[agentconfigport.StoredInstance, aggregate.Validation, agentconfigport.StoredRevision](t)
				current := storedHR(t, tt.current, mustHR(t, hr), active)
				tx.On("Instance", mock.Anything).Return(current, nil).Once()
				if tt.wantErr == nil {
					tx.On("SaveInstance", mock.Anything, mock.Anything).Return(func(_ context.Context, stored agentconfigport.StoredInstance) error {
						require.NotNil(t, stored.HR)
						snap := stored.HR.Snapshot()
						require.Equal(t, tt.current+1, snap.DraftVersion.Number())
						require.NotNil(t, snap.Active)
						require.Equal(t, revisionID, snap.Active.ID)
						return nil
					}).Once()
				}
				return fn(tx)
			}).Once()
			got, err := (agentconfigusecase.Replace{Authority: authority, Instances: instances, Clock: clock}).Execute(context.Background(), agentconfigusecase.ReplaceInput{
				InstanceID: instanceID, ExpectedDraftVersion: tt.expected, HR: &hr,
			})
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				assert.Nil(t, got)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, got)
			assert.Equal(t, tt.current+1, got.DraftVersion)
			require.NotNil(t, got.ActiveRevisionID)
			assert.Equal(t, revisionID, *got.ActiveRevisionID)
			require.NotNil(t, got.Configuration.HR.CompensationPolicy)
			assert.True(t, got.Configuration.HR.CompensationPolicy.SalaryCeilingConfigured)
		})
	}
}
