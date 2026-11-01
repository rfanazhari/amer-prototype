package agentbindingusecase_test

import (
	"context"
	"testing"

	"github.com/rfanazhari/amer-prototype/application/repository/accessport"
	"github.com/rfanazhari/amer-prototype/application/repository/agentbindingport"
	"github.com/rfanazhari/amer-prototype/application/repository/agentconfigport"
	"github.com/rfanazhari/amer-prototype/application/service"
	"github.com/rfanazhari/amer-prototype/application/usecase/agentbindingusecase"
	catalog "github.com/rfanazhari/amer-prototype/domain/agentconfig"
	aggregate "github.com/rfanazhari/amer-prototype/domain/aggregate/agentconfig"
	"github.com/rfanazhari/amer-prototype/domain/enum"
	"github.com/rfanazhari/amer-prototype/domain/valueobject"
	"github.com/rfanazhari/amer-prototype/test/mocks"
	"github.com/rfanazhari/amer-prototype/test/testdata"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// createAssignmentCase names one HR assignment-create outcome.
type createAssignmentCase struct {
	name        string
	actor       accessport.AccountRecord
	candidate   error
	instance    agentconfigport.StoredInstance
	instanceErr error
	wantErr     error
	wantWrite   bool
	wantRev     string
}

// TestCreateAssignmentPinsCurrentHRActiveRevision covers T13 CORE-ACP-013 pin selection.
func TestCreateAssignmentPinsCurrentHRActiveRevision(t *testing.T) {
	admin := testdata.AdministrativeAccountRecord(t)
	hr := testdata.AdministrativeAccountRecord(t)
	hr.Role = enum.HR
	session := testdata.AdministrativeSessionRecord(t)
	tests := []createAssignmentCase{
		{name: "hr pins current hr interviewer revision", actor: hr, instance: storedHRActive(t, firstRevision), wantWrite: true, wantRev: firstRevision},
		{name: "admin cannot create assignment", actor: admin, wantErr: service.ErrForbidden},
		{name: "unknown candidate", actor: hr, candidate: accessport.ErrNotFound, instance: storedHRActive(t, firstRevision), wantErr: accessport.ErrNotFound},
		{name: "unpublished hr interviewer", actor: hr, instance: storedHRActive(t, ""), wantErr: agentbindingport.ErrNoActiveRevision},
		{name: "document analyzer instance is rejected", actor: hr, instance: instanceWithActive(t, firstRevision), wantErr: agentbindingport.ErrUnsupportedTemplate},
		{name: "unknown instance", actor: hr, instanceErr: agentconfigport.ErrNotFound, wantErr: agentconfigport.ErrNotFound},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			candidates := mocks.NewCandidates(t)
			reader := mocks.NewReader[agentconfigport.StoredInstance, agentconfigport.StoredRevision](t)
			assignments := mocks.NewAssignments(t)
			current := mocks.NewCurrentAdministrativeSession(t)
			clock := mocks.NewClock(t)
			actorSession := session
			actorSession.AccountID = tt.actor.ID
			current.On("Resolve", mock.Anything).Return(actorSession, tt.actor, nil).Maybe()
			clock.On("Now").Return(testdata.Now()).Maybe()
			if tt.wantErr != service.ErrForbidden {
				if tt.candidate != nil {
					candidates.On("FindByID", mock.Anything, testdata.CandidateID).Return(accessport.CandidateRecord{}, tt.candidate).Maybe()
				} else {
					candidates.On("FindByID", mock.Anything, testdata.CandidateID).Return(testdata.CandidateRecord(t), nil).Maybe()
					if tt.instanceErr != nil {
						reader.On("FindInstance", mock.Anything, instanceID).Return(agentconfigport.StoredInstance{}, tt.instanceErr).Maybe()
					} else {
						reader.On("FindInstance", mock.Anything, instanceID).Return(tt.instance, nil).Maybe()
					}
				}
			}
			if tt.wantWrite {
				assignments.On("Insert", mock.Anything, mock.MatchedBy(func(binding agentbindingport.AssignmentBinding) bool {
					return binding.AssignmentID != "" &&
						binding.CandidateID == testdata.CandidateID &&
						binding.InstanceID == instanceID &&
						binding.RevisionID == tt.wantRev
				})).Return(nil).Once()
			}
			got, err := (agentbindingusecase.CreateAssignment{
				Current:     current,
				Clock:       clock,
				Candidates:  candidates,
				Instances:   reader,
				Assignments: assignments,
			}).Execute(context.Background(), testdata.CandidateID, instanceID)
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				assert.Nil(t, got)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, got)
			assert.NotEmpty(t, got.ID)
			assert.Equal(t, testdata.CandidateID, got.CandidateID)
			assert.Equal(t, instanceID, got.InstanceID)
			assert.Equal(t, tt.wantRev, got.PublishedRevisionID)
		})
	}
}

// storedHRActive restores an HR Interviewer instance with optional active revisionID.
func storedHRActive(t *testing.T, revisionID string) agentconfigport.StoredInstance {
	t.Helper()
	cfg, err := valueobject.NewHRConfiguration(valueobject.HRInput{})
	require.NoError(t, err)
	template := aggregate.TemplateRef{Template: catalog.TemplateHRInterviewer, Version: catalog.TemplateVersion1, Behavior: catalog.BehaviorVersion1}
	var active *aggregate.RevisionIdentity
	if revisionID != "" {
		active = &aggregate.RevisionIdentity{
			ID: revisionID, InstanceID: instanceID, Number: 1, DraftVersion: mustVersion(t, 1), Template: template,
		}
	}
	instance, err := aggregate.RestoreInstance(aggregate.InstanceSnapshot[valueobject.HRConfiguration]{
		ID: instanceID, Template: template, DraftVersion: mustVersion(t, 1), Configuration: cfg,
		Active: active, CreatedAt: testdata.Now(), UpdatedAt: testdata.Now(),
	})
	require.NoError(t, err)
	return agentconfigport.StoredInstance{Name: "Backend interviewer", HR: instance}
}
