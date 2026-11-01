package agentconfigusecase_test

import (
	"context"
	"errors"
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

// publishCase names one publication, replay or commit-fault outcome.
type publishCase struct {
	name                string
	txFound             bool
	txFingerprint       string
	withinErr           error
	wantReplayAuthorize bool
	wantInsert          bool
	wantFindRevision    bool
	reconcileFound      bool
	reconcileErr        error
	wantErr             error
}

// TestPublishReplaysBeforeEligibilityAndWithholdsResultsUntilCommit covers HR publication fault paths.
func TestPublishReplaysBeforeEligibilityAndWithholdsResultsUntilCommit(t *testing.T) {
	fingerprint := expectedFingerprint(instanceID, 3, validationID)
	original := agentconfigport.CommandResult{Fingerprint: fingerprint, InstanceID: instanceID, RevisionID: revisionID}
	laterActive := &aggregate.RevisionIdentity{ID: laterRevisionID, InstanceID: instanceID, Number: 2, DraftVersion: mustVersion(t, 4), Template: hrTemplate()}
	persistErr := errors.New("publication persist failed")
	uncertain := &agentconfigport.CommitUncertainError{Cause: context.DeadlineExceeded}
	tests := []publishCase{
		{name: "hr allow new publication", wantInsert: true},
		{name: "mismatched intent conflicts", txFound: true, txFingerprint: expectedFingerprint(instanceID, 4, validationID), wantErr: service.ErrIdempotencyConflict},
		{name: "old command replay after later publication", txFound: true, txFingerprint: fingerprint, wantReplayAuthorize: true, wantFindRevision: true},
		{name: "no result before commit", wantInsert: true, withinErr: persistErr, wantErr: persistErr},
		{name: "ambiguous commit reconciles original result", wantInsert: true, withinErr: uncertain, wantFindRevision: true, wantReplayAuthorize: true, reconcileFound: true},
		{name: "ambiguous commit still unavailable", wantInsert: true, withinErr: uncertain, reconcileErr: errors.New("replay store unavailable"), wantErr: uncertain},
		{name: "duplicate command reconciles outside transaction", wantInsert: true, withinErr: agentconfigport.ErrDuplicateCommand, wantFindRevision: true, wantReplayAuthorize: true, reconcileFound: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			authority := mocks.NewAgentConfigurationAuthority(t)
			instances := mocks.NewMutation[agentconfigport.StoredInstance, aggregate.Validation, agentconfigport.StoredRevision](t)
			reader := mocks.NewReader[agentconfigport.StoredInstance, agentconfigport.StoredRevision](t)
			replay := mocks.NewReplayReader(t)
			clock := mocks.NewClock(t)
			authority.On("Principal", mock.Anything).Return(hrPrincipal(), nil).Once()
			authority.On("Authorize", mock.Anything, hrPrincipal(), service.AgentConfigurationPublish, instanceID).Return(nil).Once()
			if tt.wantReplayAuthorize {
				authority.On("Authorize", mock.Anything, hrPrincipal(), service.AgentConfigurationReplay, instanceID).Return(nil).Once()
			}
			if tt.wantInsert {
				clock.On("Now").Return(testdata.Now()).Once()
			}
			instances.On("WithinInstance", mock.Anything, instanceID, mock.Anything).Return(func(ctx context.Context, _ string, fn func(agentconfigport.InstanceTx) error) error {
				tx := mocks.NewTransaction[agentconfigport.StoredInstance, aggregate.Validation, agentconfigport.StoredRevision](t)
				tx.On("FindCommand", mock.Anything, commandIdentity()).Return(agentconfigport.CommandResult{Fingerprint: tt.txFingerprint, InstanceID: instanceID, RevisionID: revisionID}, tt.txFound, nil).Once()
				if tt.wantInsert && !tt.txFound {
					tx.On("Instance", mock.Anything).Return(storedHR(t, 3, mustHR(t, validHRInput()), nil), nil).Once()
					tx.On("FindValidation", mock.Anything, validationID).Return(eligibleValidation(t, 3), nil).Once()
					tx.On("InsertRevision", mock.Anything, mock.Anything).Return(nil).Once()
					tx.On("SaveInstance", mock.Anything, mock.Anything).Return(nil).Once()
					tx.On("InsertCommand", mock.Anything, mock.MatchedBy(func(id agentconfigport.CommandIdentity) bool {
						return id == commandIdentity()
					}), mock.MatchedBy(func(result agentconfigport.CommandResult) bool {
						return result.Fingerprint == fingerprint && result.InstanceID == instanceID && result.RevisionID != ""
					})).Return(nil).Once()
				}
				callbackErr := fn(tx)
				if tt.withinErr != nil {
					return tt.withinErr
				}
				return callbackErr
			}).Once()
			if tt.withinErr != nil && !errors.Is(tt.withinErr, persistErr) {
				replay.On("FindCommand", mock.Anything, commandIdentity()).Return(original, tt.reconcileFound, tt.reconcileErr).Once()
			}
			if tt.wantFindRevision {
				reader.On("FindRevision", mock.Anything, instanceID, revisionID).Return(storedHRRevision(t, revisionID, 1, 3, mustHR(t, validHRInput())), nil).Once()
			}
			got, err := (agentconfigusecase.Publish{
				Authority: authority,
				Instances: instances,
				Reader:    reader,
				Replay:    replay,
				Clock:     clock,
			}).Execute(context.Background(), agentconfigusecase.PublishInput{
				InstanceID: instanceID, ValidationID: validationID, IdempotencyKey: publishKey, ExpectedDraftVersion: 3,
			})
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				assert.Nil(t, got)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, got)
			assert.Equal(t, instanceID, got.InstanceID)
			assert.Equal(t, validationID, got.ValidationID)
			assert.Equal(t, testdata.Now(), got.PublishedAt)
			require.NotNil(t, got.Configuration.HR)
			require.NotNil(t, got.Configuration.HR.CompensationPolicy)
			assert.True(t, got.Configuration.HR.CompensationPolicy.SalaryCeilingConfigured)
			assert.NotContains(t, got.Configuration.HR.CompensationPolicy.DiscussionMode, salaryCeilingID)
			if tt.wantFindRevision {
				assert.Equal(t, revisionID, got.RevisionID)
				assert.Equal(t, uint64(1), got.RevisionNumber)
			}
			if tt.name == "old command replay after later publication" {
				assert.NotEqual(t, laterActive.ID, got.RevisionID)
			}
		})
	}
}
