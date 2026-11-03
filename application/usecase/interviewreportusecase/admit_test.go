package interviewreportusecase_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/rfanazhari/amer-prototype/application/repository/interviewreportport"
	"github.com/rfanazhari/amer-prototype/application/service"
	"github.com/rfanazhari/amer-prototype/application/usecase/interviewreportusecase"
	"github.com/rfanazhari/amer-prototype/domain/aggregate/interviewreport"
	"github.com/rfanazhari/amer-prototype/test/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

const (
	assignmentID = "11111111-1111-4111-8111-111111111111"
	executionID  = "22222222-2222-4222-8222-222222222222"
	reportID     = "33333333-3333-4333-8333-333333333333"
	generationID = "44444444-4444-4444-8444-444444444444"
	jobID        = "55555555-5555-4555-8555-555555555555"
	outboxID     = "66666666-6666-4666-8666-666666666666"
	existingGen  = "77777777-7777-4777-8777-777777777777"
)

// fixedNow carries sub-millisecond noise and a non-UTC zone to prove the trusted instant is canonicalized.
var fixedNow = time.Date(2026, 11, 3, 10, 0, 0, 123456789, time.FixedZone("WIB", 7*3600))

// trustedNow is fixedNow as the application must hand it on: UTC at millisecond precision.
func trustedNow() time.Time { return fixedNow.UTC().Truncate(time.Millisecond) }

// admitFixture wires Admit to strict mocks.
type admitFixture struct {
	tx         *mocks.InterviewReportTransaction
	completion *mocks.InterviewReportCompletionMarker
	reports    *mocks.InterviewReportReportReader
	admissions *mocks.InterviewReportAdmissionWriter
	admit      interviewreportusecase.Admit
}

// newAdmitFixture builds an Admit whose transaction runs the callback and returns its error.
func newAdmitFixture(t *testing.T) admitFixture {
	t.Helper()
	clock := mocks.NewClock(t)
	clock.On("Now").Return(fixedNow).Maybe()
	f := admitFixture{
		tx:         mocks.NewInterviewReportTransaction(t),
		completion: mocks.NewInterviewReportCompletionMarker(t),
		reports:    mocks.NewInterviewReportReportReader(t),
		admissions: mocks.NewInterviewReportAdmissionWriter(t),
	}
	f.tx.On("InTransaction", mock.Anything, mock.Anything).Return(
		func(ctx context.Context, fn func(context.Context) error) error { return fn(ctx) }).Maybe()
	f.admit = interviewreportusecase.Admit{
		Transactions: f.tx, Completion: f.completion, Reports: f.reports, Admissions: f.admissions,
		Clock: clock, NewID: sequence(reportID, generationID, jobID, outboxID),
	}
	return f
}

// sequence returns an identity minter that yields ids in order.
func sequence(ids ...string) func() string {
	next := 0
	return func() string {
		id := ids[next]
		next++
		return id
	}
}

// completedAt is a completion marker earlier than the admission instant.
func completedAt() *time.Time {
	at := trustedNow().Add(-time.Second)
	return &at
}

// TestAdmitInsertsFirstGenerationJobAndOutbox covers the first admission of a completed interview.
func TestAdmitInsertsFirstGenerationJobAndOutbox(t *testing.T) {
	f := newAdmitFixture(t)
	f.completion.On("CompletedAt", mock.Anything, assignmentID, executionID).Return(completedAt(), nil)
	f.reports.On("FindByExecution", mock.Anything, executionID).Return(interviewreport.Snapshot{}, interviewreportport.ErrNotFound)
	var got interviewreportport.Admission
	f.admissions.On("Insert", mock.Anything, mock.Anything).Return(
		func(_ context.Context, a interviewreportport.Admission) error {
			got = a
			return nil
		})

	view, err := f.admit.Execute(context.Background(), interviewreportusecase.AdmitInput{AssignmentID: assignmentID, ExecutionID: executionID})

	require.NoError(t, err)
	require.NotNil(t, view)
	assert.Equal(t, generationID, view.ID)
	assert.Equal(t, 1, view.Number)
	assert.Equal(t, "pending", view.State)
	assert.Nil(t, view.FailureReason)
	assert.Equal(t, trustedNow(), view.CreatedAt)

	assert.Equal(t, jobID, got.JobID)
	assert.Equal(t, outboxID, got.OutboxID)
	assert.Equal(t, trustedNow().Add(interviewreport.EvidenceWait), got.WaitDeadlineAt)
	assert.Equal(t, reportID, got.Report.ID)
	assert.Equal(t, executionID, got.Report.ExecutionID)
	assert.Equal(t, assignmentID, got.Report.AssignmentID)
	assert.Equal(t, trustedNow(), got.Report.CreatedAt)
	require.Len(t, got.Report.Generations, 1)
	assert.Equal(t, generationID, got.Report.Generations[0].ID)
	assert.Equal(t, interviewreport.StatePending, got.Report.Generations[0].State)
}

// TestAdmitRefusesWithoutInsert covers every refusal that must leave the caller's transaction free of report rows.
func TestAdmitRefusesWithoutInsert(t *testing.T) {
	tests := []struct {
		name        string
		input       interviewreportusecase.AdmitInput
		completion  func(*mocks.InterviewReportCompletionMarker)
		reports     func(*mocks.InterviewReportReportReader)
		expectedErr error
		wantErr     bool
	}{
		{
			name:  "completion marker is still null",
			input: interviewreportusecase.AdmitInput{AssignmentID: assignmentID, ExecutionID: executionID},
			completion: func(m *mocks.InterviewReportCompletionMarker) {
				m.On("CompletedAt", mock.Anything, assignmentID, executionID).Return((*time.Time)(nil), nil)
			},
			expectedErr: service.ErrReportNotCompleted,
			wantErr:     true,
		},
		{
			name:  "assignment and execution do not match",
			input: interviewreportusecase.AdmitInput{AssignmentID: assignmentID, ExecutionID: executionID},
			completion: func(m *mocks.InterviewReportCompletionMarker) {
				m.On("CompletedAt", mock.Anything, assignmentID, executionID).Return((*time.Time)(nil), interviewreportport.ErrNotFound)
			},
			expectedErr: interviewreportport.ErrNotFound,
			wantErr:     true,
		},
		{
			name:  "completion read fails",
			input: interviewreportusecase.AdmitInput{AssignmentID: assignmentID, ExecutionID: executionID},
			completion: func(m *mocks.InterviewReportCompletionMarker) {
				m.On("CompletedAt", mock.Anything, assignmentID, executionID).Return((*time.Time)(nil), interviewreportport.ErrDependencyUnavailable)
			},
			expectedErr: interviewreportport.ErrDependencyUnavailable,
			wantErr:     true,
		},
		{
			name:  "report lookup fails",
			input: interviewreportusecase.AdmitInput{AssignmentID: assignmentID, ExecutionID: executionID},
			completion: func(m *mocks.InterviewReportCompletionMarker) {
				m.On("CompletedAt", mock.Anything, assignmentID, executionID).Return(completedAt(), nil)
			},
			reports: func(m *mocks.InterviewReportReportReader) {
				m.On("FindByExecution", mock.Anything, executionID).Return(interviewreport.Snapshot{}, interviewreportport.ErrDependencyUnavailable)
			},
			expectedErr: interviewreportport.ErrDependencyUnavailable,
			wantErr:     true,
		},
		{name: "empty assignment id", input: interviewreportusecase.AdmitInput{ExecutionID: executionID}, expectedErr: service.ErrReportInvalidInput, wantErr: true},
		{name: "empty execution id", input: interviewreportusecase.AdmitInput{AssignmentID: assignmentID}, expectedErr: service.ErrReportInvalidInput, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newAdmitFixture(t)
			if tt.completion != nil {
				tt.completion(f.completion)
			}
			if tt.reports != nil {
				tt.reports(f.reports)
			}

			view, err := f.admit.Execute(context.Background(), tt.input)

			assert.Equal(t, tt.wantErr, err != nil)
			assert.ErrorIs(t, err, tt.expectedErr)
			assert.Nil(t, view)
			f.admissions.AssertNotCalled(t, "Insert", mock.Anything, mock.Anything)
		})
	}
}

// TestAdmitReturnsExistingFirstGenerationWithoutSecondInsert covers a repeated call for an admitted execution.
func TestAdmitReturnsExistingFirstGenerationWithoutSecondInsert(t *testing.T) {
	f := newAdmitFixture(t)
	created := trustedNow().Add(-time.Minute)
	r, err := interviewreport.NewReport(reportID, executionID, assignmentID, created)
	require.NoError(t, err)
	_, err = r.StartGeneration(existingGen, created)
	require.NoError(t, err)
	f.completion.On("CompletedAt", mock.Anything, assignmentID, executionID).Return(completedAt(), nil)
	f.reports.On("FindByExecution", mock.Anything, executionID).Return(r.Snapshot(), nil)

	view, err := f.admit.Execute(context.Background(), interviewreportusecase.AdmitInput{AssignmentID: assignmentID, ExecutionID: executionID})

	require.NoError(t, err)
	require.NotNil(t, view)
	assert.Equal(t, existingGen, view.ID)
	assert.Equal(t, 1, view.Number)
	assert.Equal(t, created, view.CreatedAt)
	f.admissions.AssertNotCalled(t, "Insert", mock.Anything, mock.Anything)
}

// TestAdmitInsertFailureFailsTheCallerTransaction proves a store failure is returned so the caller rolls back completion.
func TestAdmitInsertFailureFailsTheCallerTransaction(t *testing.T) {
	storeErr := errors.New("outbox insert failed")
	f := newAdmitFixture(t)
	f.completion.On("CompletedAt", mock.Anything, assignmentID, executionID).Return(completedAt(), nil)
	f.reports.On("FindByExecution", mock.Anything, executionID).Return(interviewreport.Snapshot{}, interviewreportport.ErrNotFound)
	f.admissions.On("Insert", mock.Anything, mock.Anything).Return(storeErr)

	view, err := f.admit.Execute(context.Background(), interviewreportusecase.AdmitInput{AssignmentID: assignmentID, ExecutionID: executionID})

	require.Error(t, err)
	assert.ErrorIs(t, err, storeErr)
	assert.Nil(t, view)
}
