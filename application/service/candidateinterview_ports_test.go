package service_test

import (
	"context"
	"testing"

	"github.com/rfanazhari/amer-prototype/application/query/candidateinterviewquery"
	"github.com/rfanazhari/amer-prototype/application/repository/candidateinterviewport"
	"github.com/rfanazhari/amer-prototype/application/service"
	"github.com/rfanazhari/amer-prototype/application/usecase/candidateinterviewusecase"
	"github.com/rfanazhari/amer-prototype/domain/aggregate/candidateinterview"
	"github.com/rfanazhari/amer-prototype/test/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// Generated mocks must satisfy every candidate-interview contract, so a signature change breaks the build.
var (
	_ service.NoticeCatalog                              = (*mocks.NoticeCatalog)(nil)
	_ service.ExecutionLifecycle                         = (*mocks.ExecutionLifecycle)(nil)
	_ service.InterviewProgress                          = (*mocks.InterviewProgress)(nil)
	_ candidateinterviewport.ConsentTransaction          = (*mocks.ConsentTransaction)(nil)
	_ candidateinterviewport.Consents                    = (*mocks.Consents)(nil)
	_ candidateinterviewport.Controllers                 = (*mocks.Controllers)(nil)
	_ candidateinterviewusecase.AcceptNoticeOperation    = (*mocks.AcceptNoticeOperation)(nil)
	_ candidateinterviewusecase.StartInterviewOperation  = (*mocks.StartInterviewOperation)(nil)
	_ candidateinterviewusecase.TakeoverOperation        = (*mocks.TakeoverOperation)(nil)
	_ candidateinterviewusecase.CancelInterviewOperation = (*mocks.CancelInterviewOperation)(nil)
	_ candidateinterviewquery.GetInterviewOperation      = (*mocks.GetInterviewOperation)(nil)
	_ candidateinterviewquery.GetExecutionOperation      = (*mocks.GetExecutionOperation)(nil)
)

// TestLifecycleMockCarriesClassifiedErrors verifies a lifecycle failure keeps its classification across the port.
func TestLifecycleMockCarriesClassifiedErrors(t *testing.T) {
	lifecycle := mocks.NewExecutionLifecycle(t)
	req := service.CommitOrFindRequest{AssignmentID: "a1", IdempotencyKey: "k1", ConsentID: "c1"}
	lifecycle.On("CommitOrFind", mock.Anything, req).Return(service.CommitOrFindResult{}, service.ErrInterviewConflict)

	_, err := lifecycle.CommitOrFind(context.Background(), req)

	assert.ErrorIs(t, err, service.ErrInterviewConflict)
}

// TestRepositoryPortsAreFreeOfStorageTypes verifies the contracts move domain snapshots, not storage models.
func TestRepositoryPortsAreFreeOfStorageTypes(t *testing.T) {
	version, err := candidateinterview.ParseNoticeVersion("notice-261102")
	require.NoError(t, err)
	consents := mocks.NewConsents(t)
	consents.On("Find", mock.Anything, "a1", version).Return(candidateinterview.ConsentSnapshot{}, candidateinterviewport.ErrNotFound)

	_, err = consents.Find(context.Background(), "a1", version)

	assert.ErrorIs(t, err, candidateinterviewport.ErrNotFound)
}
