package agentbindingusecase_test

import (
	"context"
	"testing"

	"github.com/rfanazhari/amer-prototype/application/repository/agentbindingport"
	"github.com/rfanazhari/amer-prototype/application/repository/agentconfigport"
	"github.com/rfanazhari/amer-prototype/application/usecase/agentbindingusecase"
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
	instanceID     = "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaa8"
	firstRevision  = "cccccccc-cccc-4ccc-8ccc-ccccccccccc8"
	secondRevision = "cccccccc-cccc-4ccc-8ccc-ccccccccccc9"
	validationID   = "bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbb8"
)

// bindingCase names one assignment or execution pin outcome after later publication.
type bindingCase struct {
	name           string
	wantAssignment string
	wantExecution  string
	wantFields     []string
	wantErr        error
}

// TestBindingPreservationPinsOriginalRevisionAcrossLaterPublication covers BR-CORE-013 consumers.
func TestBindingPreservationPinsOriginalRevisionAcrossLaterPublication(t *testing.T) {
	tests := []bindingCase{
		{
			name:           "assignment pinned before later publication keeps original revision",
			wantAssignment: firstRevision,
			wantExecution:  firstRevision,
			wantFields:     []string{"total"},
		},
		{
			name:           "intentional new assignment selects the later active revision",
			wantAssignment: secondRevision,
			wantFields:     []string{"total", "currency"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := mocks.NewReader[agentconfigport.StoredInstance, agentconfigport.StoredRevision](t)
			assignments := mocks.NewAssignments(t)
			executions := mocks.NewExecutions(t)
			first := storedDocument(t, firstRevision, []string{"total"})
			second := storedDocument(t, secondRevision, []string{"total", "currency"})
			if tt.wantAssignment == firstRevision {
				reader.On("FindInstance", mock.Anything, instanceID).Return(instanceWithActive(t, firstRevision), nil).Once()
				assignments.On("Insert", mock.Anything, agentbindingport.AssignmentBinding{
					AssignmentID: testdata.AssignmentID, CandidateID: testdata.CandidateID, InstanceID: instanceID, RevisionID: firstRevision,
				}).Return(nil).Once()
				assignments.On("Find", mock.Anything, testdata.AssignmentID).Return(agentbindingport.AssignmentBinding{
					AssignmentID: testdata.AssignmentID, CandidateID: testdata.CandidateID, InstanceID: instanceID, RevisionID: firstRevision,
				}, nil).Once()
				reader.On("FindRevision", mock.Anything, instanceID, firstRevision).Return(first, nil)
				executions.On("Insert", mock.Anything, agentbindingport.ExecutionBinding{
					ExecutionID: testdata.ExecutionID, AssignmentID: testdata.AssignmentID, InstanceID: instanceID, RevisionID: firstRevision,
				}).Return(nil).Once()
				executions.On("Find", mock.Anything, testdata.ExecutionID).Return(agentbindingport.ExecutionBinding{
					ExecutionID: testdata.ExecutionID, AssignmentID: testdata.AssignmentID, InstanceID: instanceID, RevisionID: firstRevision,
				}, nil).Once()
			} else {
				reader.On("FindInstance", mock.Anything, instanceID).Return(instanceWithActive(t, secondRevision), nil).Once()
				assignments.On("Insert", mock.Anything, agentbindingport.AssignmentBinding{
					AssignmentID: testdata.AssignmentID, CandidateID: testdata.CandidateID, InstanceID: instanceID, RevisionID: secondRevision,
				}).Return(nil).Once()
			}
			pin := agentbindingusecase.PinAssignment{Instances: reader, Assignments: assignments}
			gotAssignment, err := pin.Execute(context.Background(), testdata.AssignmentID, testdata.CandidateID, instanceID)
			require.NoError(t, err)
			assert.Equal(t, tt.wantAssignment, gotAssignment.RevisionID)
			if tt.wantExecution == "" {
				return
			}
			started, err := (agentbindingusecase.StartExecution{Assignments: assignments, Executions: executions, Revisions: reader}).
				Execute(context.Background(), testdata.ExecutionID, testdata.AssignmentID)
			require.NoError(t, err)
			assert.Equal(t, tt.wantExecution, started.RevisionID)
			requirements, err := (agentbindingusecase.LoadRequirements{Executions: executions, Revisions: reader}).
				Execute(context.Background(), testdata.ExecutionID)
			require.NoError(t, err)
			assert.Equal(t, firstRevision, requirements.RevisionID)
			assert.Equal(t, tt.wantFields, requirements.RequiredFields)
			_ = second
		})
	}
}

// TestPinAssignmentRejectsUnpublishedInstance prevents binding without an active revision.
func TestPinAssignmentRejectsUnpublishedInstance(t *testing.T) {
	tests := []bindingCase{{name: "unpublished instance", wantErr: agentbindingport.ErrNoActiveRevision}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := mocks.NewReader[agentconfigport.StoredInstance, agentconfigport.StoredRevision](t)
			assignments := mocks.NewAssignments(t)
			reader.On("FindInstance", mock.Anything, instanceID).Return(unpublishedDocument(t), nil).Once()
			_, err := (agentbindingusecase.PinAssignment{Instances: reader, Assignments: assignments}).
				Execute(context.Background(), testdata.AssignmentID, testdata.CandidateID, instanceID)
			require.ErrorIs(t, err, tt.wantErr)
		})
	}
}

// mustVersion constructs a positive draft version or fails the test.
func mustVersion(t *testing.T, n uint64) aggregate.DraftVersion {
	t.Helper()
	v, err := aggregate.NewDraftVersion(n)
	require.NoError(t, err)
	return v
}

// documentTemplate is the catalog Document Analyzer v1 reference.
func documentTemplate() aggregate.TemplateRef {
	return aggregate.TemplateRef{Template: catalog.TemplateDocumentAnalyzer, Version: catalog.TemplateVersion1, Behavior: catalog.BehaviorVersion1}
}

// unpublishedDocument restores a Document instance with no active revision.
func unpublishedDocument(t *testing.T) agentconfigport.StoredInstance {
	t.Helper()
	cfg, err := valueobject.NewDocumentConfiguration([]string{"total"})
	require.NoError(t, err)
	instance, err := aggregate.RestoreInstance(aggregate.InstanceSnapshot[valueobject.DocumentConfiguration]{
		ID: instanceID, Template: documentTemplate(), DraftVersion: mustVersion(t, 1), Configuration: cfg,
		CreatedAt: testdata.Now(), UpdatedAt: testdata.Now(),
	})
	require.NoError(t, err)
	return agentconfigport.StoredInstance{Name: "Invoice reviewer", Document: instance}
}

// instanceWithActive restores a Document instance whose active pointer is revisionID.
func instanceWithActive(t *testing.T, revisionID string) agentconfigport.StoredInstance {
	t.Helper()
	cfg, err := valueobject.NewDocumentConfiguration([]string{"total"})
	require.NoError(t, err)
	identity := aggregate.RevisionIdentity{
		ID: revisionID, InstanceID: instanceID, Number: 1, DraftVersion: mustVersion(t, 1), Template: documentTemplate(),
	}
	if revisionID == secondRevision {
		identity.Number = 2
		identity.DraftVersion = mustVersion(t, 2)
		cfg, err = valueobject.NewDocumentConfiguration([]string{"total", "currency"})
		require.NoError(t, err)
	}
	instance, err := aggregate.RestoreInstance(aggregate.InstanceSnapshot[valueobject.DocumentConfiguration]{
		ID: instanceID, Template: documentTemplate(), DraftVersion: identity.DraftVersion, Configuration: cfg,
		Active: &identity, CreatedAt: testdata.Now(), UpdatedAt: testdata.Now(),
	})
	require.NoError(t, err)
	return agentconfigport.StoredInstance{Name: "Invoice reviewer", Document: instance}
}

// storedDocument restores an immutable Document revision with the given required fields.
func storedDocument(t *testing.T, revisionID string, fields []string) agentconfigport.StoredRevision {
	t.Helper()
	cfg, err := valueobject.NewDocumentConfiguration(fields)
	require.NoError(t, err)
	number := uint64(1)
	draft := mustVersion(t, 1)
	if revisionID == secondRevision {
		number = 2
		draft = mustVersion(t, 2)
	}
	revision, err := aggregate.RestoreRevision(aggregate.RevisionSnapshot[valueobject.DocumentConfiguration]{
		Identity: aggregate.RevisionIdentity{
			ID: revisionID, InstanceID: instanceID, Number: number, DraftVersion: draft, Template: documentTemplate(),
		},
		ValidationID: validationID, Configuration: cfg, CreatedAt: testdata.Now(),
	})
	require.NoError(t, err)
	return agentconfigport.StoredRevision{Document: &revision}
}
