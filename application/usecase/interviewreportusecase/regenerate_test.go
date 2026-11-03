package interviewreportusecase_test

import (
	"context"
	"errors"
	"strings"
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
	commandID = "88888888-8888-4888-8888-888888888888"
	subjectID = "99999999-9999-4999-8999-999999999999"
	secondGen = "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa"
	requestID = "regen-2026-11-03"
	// regenerateFingerprint is the SHA-256 of the compact JSON array ["regenerate-interview-report/v1","<assignmentID>"],
	// computed outside the implementation.
	regenerateFingerprint = "8e95c780ce86a54d3bae4ae8c3550bf10c04f2e2e21efe1731d43db2d126e6fc"
)

var (
	errDenied  = errors.New("denied")
	principal  = service.AdminPrincipal{SubjectID: subjectID}
	identity   = interviewreportport.CommandIdentity{Scope: subjectID, Kind: interviewreportusecase.RegenerateCommandKind, Key: requestID}
	errStore   = errors.New("store failed")
	regenInput = interviewreportusecase.RegenerateInput{AssignmentID: assignmentID, IdempotencyKey: requestID}
)

// regenerateFixture wires Regenerate to strict mocks.
type regenerateFixture struct {
	authority   *mocks.InterviewReportAuthority
	assignments *mocks.InterviewReportAssignmentLocker
	reports     *mocks.InterviewReportReportReader
	commands    *mocks.InterviewReportRegenerationStore
	evidence    *mocks.InterviewReportEvidence
	regenerate  interviewreportusecase.Regenerate
}

// newRegenerateFixture builds a Regenerate whose transaction runs the callback and returns its error.
func newRegenerateFixture(t *testing.T, ids ...string) regenerateFixture {
	t.Helper()
	clock := mocks.NewClock(t)
	clock.On("Now").Return(fixedNow).Maybe()
	tx := mocks.NewInterviewReportTransaction(t)
	tx.On("InTransaction", mock.Anything, mock.Anything).Return(
		func(ctx context.Context, fn func(context.Context) error) error { return fn(ctx) }).Maybe()
	f := regenerateFixture{
		authority:   mocks.NewInterviewReportAuthority(t),
		assignments: mocks.NewInterviewReportAssignmentLocker(t),
		reports:     mocks.NewInterviewReportReportReader(t),
		commands:    mocks.NewInterviewReportRegenerationStore(t),
		evidence:    mocks.NewInterviewReportEvidence(t),
	}
	f.regenerate = interviewreportusecase.Regenerate{
		Authority: f.authority, Transactions: tx, Assignments: f.assignments, Reports: f.reports,
		Commands: f.commands, Evidence: f.evidence, Clock: clock, NewID: sequence(ids...),
	}
	return f
}

// allowHR makes the authority accept the regeneration of the assignment for the HR principal.
func (f regenerateFixture) allowHR() {
	f.authority.On("Principal", mock.Anything).Return(principal, nil)
	f.authority.On("Authorize", mock.Anything, principal, service.InterviewReportRegenerate, mock.Anything).Return(nil)
}

// lockCompleted makes the assignment lock return a completed interview.
func (f regenerateFixture) lockCompleted() {
	f.assignments.On("LockAssignment", mock.Anything, mock.Anything).Return(
		interviewreportport.AssignmentRef{AssignmentID: assignmentID, ExecutionID: executionID, CompletedAt: completedAt()}, nil)
}

// noCommand makes the command lookup report no stored command.
func (f regenerateFixture) noCommand() {
	f.commands.On("FindCommand", mock.Anything, identity).Return(interviewreportport.StoredCommand{}, false, nil).Once()
}

// finalized makes the evidence port list one finalized entry.
func (f regenerateFixture) finalized() {
	f.evidence.On("Finalized", mock.Anything, executionID).Return(
		[]interviewreportport.EvidenceEntry{{ID: "entry-1", Revision: 1}}, nil)
}

// failedSnapshot is a report whose only generation failed.
func failedSnapshot(t *testing.T) interviewreport.Snapshot {
	t.Helper()
	created := trustedNow().Add(-time.Hour)
	r, err := interviewreport.NewReport(reportID, executionID, assignmentID, created)
	require.NoError(t, err)
	_, err = r.StartGeneration(existingGen, created)
	require.NoError(t, err)
	_, err = r.Fail(existingGen, "evidence_unavailable", created.Add(time.Minute))
	require.NoError(t, err)
	return r.Snapshot()
}

// activeSnapshot is a report whose second generation is still pending after the first failed.
func activeSnapshot(t *testing.T) interviewreport.Snapshot {
	t.Helper()
	r, err := interviewreport.RestoreReport(failedSnapshot(t))
	require.NoError(t, err)
	_, err = r.StartGeneration(secondGen, trustedNow().Add(-time.Minute))
	require.NoError(t, err)
	return r.Snapshot()
}

// pendingSnapshot is a report whose first generation is still pending.
func pendingSnapshot(t *testing.T) interviewreport.Snapshot {
	t.Helper()
	created := trustedNow().Add(-time.Minute)
	r, err := interviewreport.NewReport(reportID, executionID, assignmentID, created)
	require.NoError(t, err)
	_, err = r.StartGeneration(existingGen, created)
	require.NoError(t, err)
	return r.Snapshot()
}

// TestRegenerateStartsNextGenerationAfterFailure covers a new key after a terminal generation.
func TestRegenerateStartsNextGenerationAfterFailure(t *testing.T) {
	f := newRegenerateFixture(t, generationID, jobID, outboxID, commandID)
	f.allowHR()
	f.lockCompleted()
	f.noCommand()
	f.reports.On("FindByExecution", mock.Anything, executionID).Return(failedSnapshot(t), nil)
	f.finalized()
	var got interviewreportport.Regeneration
	f.commands.On("InsertRegeneration", mock.Anything, mock.Anything).Return(
		func(_ context.Context, r interviewreportport.Regeneration) error {
			got = r
			return nil
		})

	result, err := f.regenerate.Execute(context.Background(), regenInput)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.False(t, result.Replayed)
	require.NotNil(t, result.Generation)
	assert.Equal(t, generationID, result.Generation.ID)
	assert.Equal(t, 2, result.Generation.Number)
	assert.Equal(t, "pending", result.Generation.State)
	assert.Nil(t, result.Generation.FailureReason)
	assert.Equal(t, trustedNow(), result.Generation.CreatedAt)

	assert.Equal(t, identity, got.Command)
	assert.Equal(t, commandID, got.CommandID)
	assert.Equal(t, regenerateFingerprint, got.Fingerprint)
	assert.False(t, got.CreatesReport)
	assert.Equal(t, jobID, got.JobID)
	assert.Equal(t, outboxID, got.OutboxID)
	assert.Equal(t, trustedNow().Add(interviewreport.EvidenceWait), got.WaitDeadlineAt)
	assert.Equal(t, reportID, got.Report.ID)
	require.Len(t, got.Report.Generations, 2)
	assert.Equal(t, interviewreport.StateFailed, got.Report.Generations[0].State)
	assert.Equal(t, generationID, got.Report.Generations[1].ID)
	assert.Equal(t, interviewreport.StatePending, got.Report.Generations[1].State)
}

// TestRegenerateCreatesReportForCompletedInterviewWithoutOne covers an explicit request for an interview that predates reporting.
func TestRegenerateCreatesReportForCompletedInterviewWithoutOne(t *testing.T) {
	f := newRegenerateFixture(t, reportID, generationID, jobID, outboxID, commandID)
	f.allowHR()
	f.lockCompleted()
	f.noCommand()
	f.reports.On("FindByExecution", mock.Anything, executionID).Return(interviewreport.Snapshot{}, interviewreportport.ErrNotFound)
	f.finalized()
	var got interviewreportport.Regeneration
	f.commands.On("InsertRegeneration", mock.Anything, mock.Anything).Return(
		func(_ context.Context, r interviewreportport.Regeneration) error {
			got = r
			return nil
		})

	result, err := f.regenerate.Execute(context.Background(), regenInput)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, 1, result.Generation.Number)
	assert.True(t, got.CreatesReport)
	assert.Equal(t, reportID, got.Report.ID)
	assert.Equal(t, executionID, got.Report.ExecutionID)
	assert.Equal(t, assignmentID, got.Report.AssignmentID)
	require.Len(t, got.Report.Generations, 1)
	assert.Equal(t, generationID, got.Report.Generations[0].ID)
	assert.Equal(t, commandID, got.CommandID)
}

// TestRegenerateReplayReturnsOriginalGeneration covers the same key and fingerprint while a later generation exists.
func TestRegenerateReplayReturnsOriginalGeneration(t *testing.T) {
	f := newRegenerateFixture(t)
	f.allowHR()
	f.lockCompleted()
	f.commands.On("FindCommand", mock.Anything, identity).Return(
		interviewreportport.StoredCommand{Fingerprint: regenerateFingerprint, GenerationID: existingGen, ExecutionID: executionID}, true, nil)
	f.reports.On("FindByExecution", mock.Anything, executionID).Return(activeSnapshot(t), nil)

	result, err := f.regenerate.Execute(context.Background(), regenInput)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, result.Replayed)
	require.NotNil(t, result.Generation)
	assert.Equal(t, existingGen, result.Generation.ID)
	assert.Equal(t, 1, result.Generation.Number)
	assert.Equal(t, "failed", result.Generation.State)
	f.commands.AssertNotCalled(t, "InsertRegeneration", mock.Anything, mock.Anything)
	f.evidence.AssertNotCalled(t, "Finalized", mock.Anything, mock.Anything)
}

// TestRegenerateFingerprintUsesLowercaseAssignment proves the fingerprint is case-insensitive in the assignment id.
func TestRegenerateFingerprintUsesLowercaseAssignment(t *testing.T) {
	upper := strings.ToUpper(assignmentID)
	f := newRegenerateFixture(t)
	f.allowHR()
	f.assignments.On("LockAssignment", mock.Anything, upper).Return(
		interviewreportport.AssignmentRef{AssignmentID: assignmentID, ExecutionID: executionID, CompletedAt: completedAt()}, nil)
	f.commands.On("FindCommand", mock.Anything, identity).Return(
		interviewreportport.StoredCommand{Fingerprint: regenerateFingerprint, GenerationID: existingGen, ExecutionID: executionID}, true, nil)
	f.reports.On("FindByExecution", mock.Anything, executionID).Return(failedSnapshot(t), nil)

	result, err := f.regenerate.Execute(context.Background(), interviewreportusecase.RegenerateInput{AssignmentID: upper, IdempotencyKey: requestID})

	require.NoError(t, err)
	assert.True(t, result.Replayed)
}

// TestRegenerateRefusesWithoutInsert covers every refusal that must leave the stores free of new rows.
func TestRegenerateRefusesWithoutInsert(t *testing.T) {
	tests := []struct {
		name        string
		setup       func(regenerateFixture, *testing.T)
		expectedErr error
		wantErr     bool
	}{
		{
			name: "assignment is unknown",
			setup: func(f regenerateFixture, _ *testing.T) {
				f.assignments.On("LockAssignment", mock.Anything, assignmentID).Return(interviewreportport.AssignmentRef{}, interviewreportport.ErrNotFound)
			},
			expectedErr: interviewreportport.ErrNotFound,
			wantErr:     true,
		},
		{
			name: "same key with a different intent",
			setup: func(f regenerateFixture, _ *testing.T) {
				f.lockCompleted()
				f.commands.On("FindCommand", mock.Anything, identity).Return(
					interviewreportport.StoredCommand{Fingerprint: strings.Repeat("0", 64), GenerationID: existingGen, ExecutionID: executionID}, true, nil)
			},
			expectedErr: service.ErrReportIdempotencyConflict,
			wantErr:     true,
		},
		{
			name: "command lookup fails",
			setup: func(f regenerateFixture, _ *testing.T) {
				f.lockCompleted()
				f.commands.On("FindCommand", mock.Anything, identity).Return(interviewreportport.StoredCommand{}, false, errStore)
			},
			expectedErr: errStore,
			wantErr:     true,
		},
		{
			name: "interview is not completed",
			setup: func(f regenerateFixture, _ *testing.T) {
				f.assignments.On("LockAssignment", mock.Anything, assignmentID).Return(
					interviewreportport.AssignmentRef{AssignmentID: assignmentID, ExecutionID: executionID}, nil)
				f.noCommand()
			},
			expectedErr: service.ErrReportNotCompleted,
			wantErr:     true,
		},
		{
			name: "evidence port lists nothing finalized",
			setup: func(f regenerateFixture, _ *testing.T) {
				f.lockCompleted()
				f.noCommand()
				f.evidence.On("Finalized", mock.Anything, executionID).Return([]interviewreportport.EvidenceEntry{}, nil)
			},
			expectedErr: service.ErrReportEvidenceUnavailable,
			wantErr:     true,
		},
		{
			name: "evidence port fails",
			setup: func(f regenerateFixture, _ *testing.T) {
				f.lockCompleted()
				f.noCommand()
				f.evidence.On("Finalized", mock.Anything, executionID).Return(nil, errStore)
			},
			expectedErr: errStore,
			wantErr:     true,
		},
		{
			name: "report lookup fails",
			setup: func(f regenerateFixture, _ *testing.T) {
				f.lockCompleted()
				f.noCommand()
				f.finalized()
				f.reports.On("FindByExecution", mock.Anything, executionID).Return(interviewreport.Snapshot{}, interviewreportport.ErrDependencyUnavailable)
			},
			expectedErr: interviewreportport.ErrDependencyUnavailable,
			wantErr:     true,
		},
		{
			name: "a generation is pending",
			setup: func(f regenerateFixture, t *testing.T) {
				f.lockCompleted()
				f.noCommand()
				f.finalized()
				f.reports.On("FindByExecution", mock.Anything, executionID).Return(pendingSnapshot(t), nil)
			},
			expectedErr: interviewreport.ErrGenerationActive,
			wantErr:     true,
		},
		{
			name: "a generation became active before insert",
			setup: func(f regenerateFixture, t *testing.T) {
				f.lockCompleted()
				f.noCommand()
				f.finalized()
				f.reports.On("FindByExecution", mock.Anything, executionID).Return(failedSnapshot(t), nil)
				f.commands.On("InsertRegeneration", mock.Anything, mock.Anything).Return(interviewreport.ErrGenerationActive)
			},
			expectedErr: interviewreport.ErrGenerationActive,
			wantErr:     true,
		},
		{
			name: "insert fails",
			setup: func(f regenerateFixture, t *testing.T) {
				f.lockCompleted()
				f.noCommand()
				f.finalized()
				f.reports.On("FindByExecution", mock.Anything, executionID).Return(failedSnapshot(t), nil)
				f.commands.On("InsertRegeneration", mock.Anything, mock.Anything).Return(errStore)
			},
			expectedErr: errStore,
			wantErr:     true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newRegenerateFixture(t, generationID, jobID, outboxID, commandID)
			f.allowHR()
			tt.setup(f, t)

			result, err := f.regenerate.Execute(context.Background(), regenInput)

			assert.Equal(t, tt.wantErr, err != nil)
			assert.ErrorIs(t, err, tt.expectedErr)
			assert.Nil(t, result)
		})
	}
}

// TestRegenerateDoesNotInsertWhenRefused proves that a refusal before the final write never reaches the insert.
func TestRegenerateDoesNotInsertWhenRefused(t *testing.T) {
	f := newRegenerateFixture(t, generationID, jobID, outboxID, commandID)
	f.allowHR()
	f.lockCompleted()
	f.noCommand()
	f.finalized()
	f.reports.On("FindByExecution", mock.Anything, executionID).Return(activeSnapshot(t), nil)

	result, err := f.regenerate.Execute(context.Background(), regenInput)

	assert.ErrorIs(t, err, interviewreport.ErrGenerationActive)
	assert.Nil(t, result)
	f.commands.AssertNotCalled(t, "InsertRegeneration", mock.Anything, mock.Anything)
}

// TestRegenerateReconcilesUniqueCommandLoss covers the loser of a concurrent request with the same key.
func TestRegenerateReconcilesUniqueCommandLoss(t *testing.T) {
	tests := []struct {
		name        string
		stored      string
		expectedErr error
		wantErr     bool
	}{
		{name: "same fingerprint replays the winner", stored: regenerateFingerprint},
		{name: "different fingerprint conflicts", stored: strings.Repeat("0", 64), expectedErr: service.ErrReportIdempotencyConflict, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newRegenerateFixture(t, generationID, jobID, outboxID, commandID)
			f.allowHR()
			f.lockCompleted()
			f.noCommand()
			f.finalized()
			f.reports.On("FindByExecution", mock.Anything, executionID).Return(failedSnapshot(t), nil).Once()
			f.commands.On("InsertRegeneration", mock.Anything, mock.Anything).Return(interviewreportport.ErrDuplicateCommand)
			f.commands.On("FindCommand", mock.Anything, identity).Return(
				interviewreportport.StoredCommand{Fingerprint: tt.stored, GenerationID: secondGen, ExecutionID: executionID}, true, nil).Once()
			f.reports.On("FindByExecution", mock.Anything, executionID).Return(activeSnapshot(t), nil).Maybe()

			result, err := f.regenerate.Execute(context.Background(), regenInput)

			assert.Equal(t, tt.wantErr, err != nil)
			assert.ErrorIs(t, err, tt.expectedErr)
			if tt.wantErr {
				assert.Nil(t, result)
				return
			}
			require.NotNil(t, result)
			assert.True(t, result.Replayed)
			assert.Equal(t, secondGen, result.Generation.ID)
			assert.Equal(t, 2, result.Generation.Number)
		})
	}
}

// TestRegenerateDeniesBeforeAnyLookup proves authorization runs before the assignment is looked up.
func TestRegenerateDeniesBeforeAnyLookup(t *testing.T) {
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
				a.On("Authorize", mock.Anything, principal, service.InterviewReportRegenerate, assignmentID).Return(errDenied)
			},
			expectedErr: errDenied,
			wantErr:     true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newRegenerateFixture(t)
			tt.setup(f.authority)

			result, err := f.regenerate.Execute(context.Background(), regenInput)

			assert.Equal(t, tt.wantErr, err != nil)
			assert.ErrorIs(t, err, tt.expectedErr)
			assert.Nil(t, result)
			f.assignments.AssertNotCalled(t, "LockAssignment", mock.Anything, mock.Anything)
		})
	}
}

// TestRegenerateRejectsInvalidInput covers a malformed key or assignment, which fails after authorization and before lookup.
func TestRegenerateRejectsInvalidInput(t *testing.T) {
	tests := []struct {
		name        string
		input       interviewreportusecase.RegenerateInput
		expectedErr error
		wantErr     bool
	}{
		{name: "empty key", input: interviewreportusecase.RegenerateInput{AssignmentID: assignmentID}, expectedErr: service.ErrReportInvalidInput, wantErr: true},
		{name: "key with surrounding space", input: interviewreportusecase.RegenerateInput{AssignmentID: assignmentID, IdempotencyKey: " " + requestID}, expectedErr: service.ErrReportInvalidInput, wantErr: true},
		{name: "key starting with punctuation", input: interviewreportusecase.RegenerateInput{AssignmentID: assignmentID, IdempotencyKey: "-abc"}, expectedErr: service.ErrReportInvalidInput, wantErr: true},
		{name: "key with a comma", input: interviewreportusecase.RegenerateInput{AssignmentID: assignmentID, IdempotencyKey: "a,b"}, expectedErr: service.ErrReportInvalidInput, wantErr: true},
		{name: "key longer than 128", input: interviewreportusecase.RegenerateInput{AssignmentID: assignmentID, IdempotencyKey: strings.Repeat("a", 129)}, expectedErr: service.ErrReportInvalidInput, wantErr: true},
		{name: "empty assignment", input: interviewreportusecase.RegenerateInput{IdempotencyKey: requestID}, expectedErr: service.ErrReportInvalidInput, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newRegenerateFixture(t)
			f.authority.On("Principal", mock.Anything).Return(principal, nil)
			f.authority.On("Authorize", mock.Anything, principal, service.InterviewReportRegenerate, mock.Anything).Return(nil)

			result, err := f.regenerate.Execute(context.Background(), tt.input)

			assert.Equal(t, tt.wantErr, err != nil)
			assert.ErrorIs(t, err, tt.expectedErr)
			assert.Nil(t, result)
			f.assignments.AssertNotCalled(t, "LockAssignment", mock.Anything, mock.Anything)
		})
	}
}
