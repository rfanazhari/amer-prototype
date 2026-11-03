package interviewreportquery_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/rfanazhari/amer-prototype/application/query/interviewreportquery"
	"github.com/rfanazhari/amer-prototype/application/repository/interviewreportport"
	"github.com/rfanazhari/amer-prototype/application/service"
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
	firstGen     = "44444444-4444-4444-8444-444444444444"
	secondGen    = "77777777-7777-4777-8777-777777777777"
	thirdGen     = "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa"
)

var (
	errDenied = errors.New("denied")
	principal = service.AdminPrincipal{SubjectID: "99999999-9999-4999-8999-999999999999"}
	created   = time.Date(2026, 11, 3, 1, 0, 0, 0, time.UTC)
)

// readFixture wires Read to strict mocks.
type readFixture struct {
	authority   *mocks.InterviewReportAuthority
	assignments *mocks.InterviewReportAssignmentReader
	reports     *mocks.InterviewReportReportReader
	read        interviewreportquery.Read
}

// newReadFixture builds a Read over strict mocks.
func newReadFixture(t *testing.T) readFixture {
	t.Helper()
	f := readFixture{
		authority:   mocks.NewInterviewReportAuthority(t),
		assignments: mocks.NewInterviewReportAssignmentReader(t),
		reports:     mocks.NewInterviewReportReportReader(t),
	}
	f.read = interviewreportquery.Read{Authority: f.authority, Assignments: f.assignments, Reports: f.reports}
	return f
}

// allowHR makes the authority accept the read for the HR principal.
func (f readFixture) allowHR() {
	f.authority.On("Principal", mock.Anything).Return(principal, nil)
	f.authority.On("Authorize", mock.Anything, principal, service.InterviewReportRead, mock.Anything).Return(nil)
}

// found makes the assignment lookup return the execution pin with the given completion marker.
func (f readFixture) found(completed *time.Time) {
	f.assignments.On("FindAssignment", mock.Anything, assignmentID).Return(
		interviewreportport.AssignmentRef{AssignmentID: assignmentID, ExecutionID: executionID, CompletedAt: completed}, nil)
}

// statement builds a validated statement and fails the test when the input is rejected.
func statement(t *testing.T, in interviewreport.StatementInput) interviewreport.Statement {
	t.Helper()
	st, err := interviewreport.NewStatement(in)
	require.NoError(t, err)
	return st
}

// historySnapshot builds a report whose first generation succeeded with a full account, whose second
// generation ended as the given state, and whose third generation is added when the second is terminal.
func historySnapshot(t *testing.T, second string) interviewreport.Snapshot {
	t.Helper()
	r, err := interviewreport.NewReport(reportID, executionID, assignmentID, created)
	require.NoError(t, err)
	_, err = r.StartGeneration(firstGen, created)
	require.NoError(t, err)
	_, err = r.Run(firstGen, created.Add(time.Second))
	require.NoError(t, err)
	statements := []interviewreport.Statement{
		statement(t, interviewreport.StatementInput{Kind: interviewreport.KindObservation, Body: "Backend experience.", PresentsAsAnswer: true, Citation: &interviewreport.Citation{EntryID: "entry-1", Revision: 1}}),
		statement(t, interviewreport.StatementInput{Kind: interviewreport.KindGap, Body: "Compensation was not answered."}),
		statement(t, interviewreport.StatementInput{Kind: interviewreport.KindUncertainty, Body: "Expectation was unclear.", Citation: &interviewreport.Citation{EntryID: "entry-2", Revision: 2}}),
		statement(t, interviewreport.StatementInput{Kind: interviewreport.KindUncertainty, Body: "Notice period unclear."}),
		statement(t, interviewreport.StatementInput{Kind: interviewreport.KindObservation, Body: "Mentioned a side project."}),
	}
	_, err = r.Succeed(firstGen, statements, created.Add(2*time.Second))
	require.NoError(t, err)
	if second == "" {
		return r.Snapshot()
	}
	_, err = r.StartGeneration(secondGen, created.Add(time.Minute))
	require.NoError(t, err)
	if second == "failed" {
		_, err = r.Fail(secondGen, "evidence_unavailable", created.Add(2*time.Minute))
		require.NoError(t, err)
	}
	return r.Snapshot()
}

// TestReadReturnsNewestGenerationWithLatestSucceededAccount covers the generation and account selection.
func TestReadReturnsNewestGenerationWithLatestSucceededAccount(t *testing.T) {
	tests := []struct {
		name               string
		second             string
		expectedGeneration string
		expectedNumber     int
		expectedState      string
		expectedFailure    *string
	}{
		{name: "only generation succeeded", second: "", expectedGeneration: firstGen, expectedNumber: 1, expectedState: "succeeded"},
		{name: "newer generation is pending", second: "pending", expectedGeneration: secondGen, expectedNumber: 2, expectedState: "pending"},
		{name: "newer generation failed", second: "failed", expectedGeneration: secondGen, expectedNumber: 2, expectedState: "failed", expectedFailure: ptr("evidence_unavailable")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newReadFixture(t)
			f.allowHR()
			f.found(&created)
			f.reports.On("FindByExecution", mock.Anything, executionID).Return(historySnapshot(t, tt.second), nil)

			view, err := f.read.Execute(context.Background(), interviewreportquery.ReadInput{AssignmentID: assignmentID})

			require.NoError(t, err)
			require.NotNil(t, view)
			assert.Equal(t, assignmentID, view.AssignmentID)
			assert.Equal(t, executionID, view.ExecutionID)
			assert.Equal(t, tt.expectedGeneration, view.Generation.ID)
			assert.Equal(t, tt.expectedNumber, view.Generation.Number)
			assert.Equal(t, tt.expectedState, view.Generation.State)
			assert.Equal(t, tt.expectedFailure, view.Generation.FailureReason)
			require.NotNil(t, view.Report)
			assert.Equal(t, firstGen, view.Report.GenerationID)
		})
	}
}

// TestReadMapsStatementsIntoTheThreeKinds covers observations, gaps and uncertainties in the account.
func TestReadMapsStatementsIntoTheThreeKinds(t *testing.T) {
	f := newReadFixture(t)
	f.allowHR()
	f.found(&created)
	f.reports.On("FindByExecution", mock.Anything, executionID).Return(historySnapshot(t, ""), nil)

	view, err := f.read.Execute(context.Background(), interviewreportquery.ReadInput{AssignmentID: assignmentID})

	require.NoError(t, err)
	require.NotNil(t, view.Report)
	require.Len(t, view.Report.Observations, 1)
	assert.Equal(t, "Backend experience.", view.Report.Observations[0].Body)
	assert.Equal(t, "entry-1", view.Report.Observations[0].EvidenceEntryID)
	assert.Equal(t, 1, view.Report.Observations[0].EvidenceRevision)
	require.Len(t, view.Report.Gaps, 1)
	assert.Equal(t, "Compensation was not answered.", view.Report.Gaps[0].Body)
	assert.Nil(t, view.Report.Gaps[0].QuestionID)
	require.Len(t, view.Report.Uncertainties, 3)
	assert.Equal(t, ptr("entry-2"), view.Report.Uncertainties[0].EvidenceEntryID)
	assert.Equal(t, ptr(2), view.Report.Uncertainties[0].EvidenceRevision)
	assert.Equal(t, "Notice period unclear.", view.Report.Uncertainties[1].Body)
	assert.Nil(t, view.Report.Uncertainties[1].EvidenceEntryID)
	assert.Nil(t, view.Report.Uncertainties[1].EvidenceRevision)
	assert.Equal(t, "Mentioned a side project.", view.Report.Uncertainties[2].Body)
}

// TestReadReturnsNullAccountUntilAGenerationSucceeds covers a report whose generations have not succeeded.
func TestReadReturnsNullAccountUntilAGenerationSucceeds(t *testing.T) {
	r, err := interviewreport.NewReport(reportID, executionID, assignmentID, created)
	require.NoError(t, err)
	_, err = r.StartGeneration(firstGen, created)
	require.NoError(t, err)
	f := newReadFixture(t)
	f.allowHR()
	f.found(&created)
	f.reports.On("FindByExecution", mock.Anything, executionID).Return(r.Snapshot(), nil)

	view, err := f.read.Execute(context.Background(), interviewreportquery.ReadInput{AssignmentID: assignmentID})

	require.NoError(t, err)
	assert.Equal(t, "pending", view.Generation.State)
	assert.Nil(t, view.Report)
}

// TestReadRefuses covers every refusal after authorization.
func TestReadRefuses(t *testing.T) {
	tests := []struct {
		name        string
		input       interviewreportquery.ReadInput
		setup       func(readFixture)
		expectedErr error
		wantErr     bool
	}{
		{
			name:        "empty assignment",
			input:       interviewreportquery.ReadInput{},
			expectedErr: service.ErrReportInvalidInput,
			wantErr:     true,
		},
		{
			name:  "assignment is unknown",
			input: interviewreportquery.ReadInput{AssignmentID: assignmentID},
			setup: func(f readFixture) {
				f.assignments.On("FindAssignment", mock.Anything, assignmentID).Return(interviewreportport.AssignmentRef{}, interviewreportport.ErrNotFound)
			},
			expectedErr: interviewreportport.ErrNotFound,
			wantErr:     true,
		},
		{
			name:  "interview is not completed and has no report",
			input: interviewreportquery.ReadInput{AssignmentID: assignmentID},
			setup: func(f readFixture) {
				f.found(nil)
				f.reports.On("FindByExecution", mock.Anything, executionID).Return(interviewreport.Snapshot{}, interviewreportport.ErrNotFound)
			},
			expectedErr: service.ErrReportNotCompleted,
			wantErr:     true,
		},
		{
			name:  "interview is completed and has no report",
			input: interviewreportquery.ReadInput{AssignmentID: assignmentID},
			setup: func(f readFixture) {
				f.found(&created)
				f.reports.On("FindByExecution", mock.Anything, executionID).Return(interviewreport.Snapshot{}, interviewreportport.ErrNotFound)
			},
			expectedErr: interviewreportport.ErrNotFound,
			wantErr:     true,
		},
		{
			name:  "report lookup fails",
			input: interviewreportquery.ReadInput{AssignmentID: assignmentID},
			setup: func(f readFixture) {
				f.found(&created)
				f.reports.On("FindByExecution", mock.Anything, executionID).Return(interviewreport.Snapshot{}, interviewreportport.ErrDependencyUnavailable)
			},
			expectedErr: interviewreportport.ErrDependencyUnavailable,
			wantErr:     true,
		},
		{
			name:  "stored report has no generation",
			input: interviewreportquery.ReadInput{AssignmentID: assignmentID},
			setup: func(f readFixture) {
				f.found(&created)
				f.reports.On("FindByExecution", mock.Anything, executionID).Return(interviewreport.Snapshot{ID: reportID, ExecutionID: executionID, AssignmentID: assignmentID, CreatedAt: created}, nil)
			},
			expectedErr: interviewreport.ErrInvalidReport,
			wantErr:     true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newReadFixture(t)
			f.allowHR()
			if tt.setup != nil {
				tt.setup(f)
			}

			view, err := f.read.Execute(context.Background(), tt.input)

			assert.Equal(t, tt.wantErr, err != nil)
			assert.ErrorIs(t, err, tt.expectedErr)
			assert.Nil(t, view)
		})
	}
}

// TestReadDeniesBeforeAnyLookup proves authorization runs before the assignment is looked up.
func TestReadDeniesBeforeAnyLookup(t *testing.T) {
	tests := []struct {
		name        string
		setup       func(*mocks.InterviewReportAuthority)
		expectedErr error
		wantErr     bool
	}{
		{
			name: "principal cannot be resolved",
			setup: func(a *mocks.InterviewReportAuthority) {
				a.On("Principal", mock.Anything).Return(service.AdminPrincipal{}, errDenied)
			},
			expectedErr: errDenied,
			wantErr:     true,
		},
		{
			name: "operation is not allowed for the principal",
			setup: func(a *mocks.InterviewReportAuthority) {
				a.On("Principal", mock.Anything).Return(principal, nil)
				a.On("Authorize", mock.Anything, principal, service.InterviewReportRead, assignmentID).Return(errDenied)
			},
			expectedErr: errDenied,
			wantErr:     true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newReadFixture(t)
			tt.setup(f.authority)

			view, err := f.read.Execute(context.Background(), interviewreportquery.ReadInput{AssignmentID: assignmentID})

			assert.Equal(t, tt.wantErr, err != nil)
			assert.ErrorIs(t, err, tt.expectedErr)
			assert.Nil(t, view)
			f.assignments.AssertNotCalled(t, "FindAssignment", mock.Anything, mock.Anything)
		})
	}
}

// ptr returns a pointer to a copy of v.
func ptr[T any](v T) *T { return &v }
