package interviewreportusecase_test

import (
	"context"
	"errors"
	"github.com/rfanazhari/amer-prototype/application/repository/interviewreportport"
	"github.com/rfanazhari/amer-prototype/application/usecase/interviewreportusecase"
	"github.com/rfanazhari/amer-prototype/domain/aggregate/interviewreport"
	"github.com/rfanazhari/amer-prototype/test/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

// TestWorkerExecute covers writer deadlines, final citations, output rejection and database error propagation.
func TestWorkerExecute(t *testing.T) {
	now := time.Date(2026, 11, 3, 0, 0, 0, 0, time.UTC)
	failure := errors.New("dependency failed")
	// testCase describes one independent public behavior scenario.
	type testCase struct {
		name                    string
		wantErr                 bool
		output                  []interviewreport.StatementInput
		writerErr               error
		final                   bool
		claimErr, errorOnFinish error
		reason                  string
	}
	tests := []testCase{
		{name: "accepted finalized answer", final: true, output: []interviewreport.StatementInput{{Kind: interviewreport.KindObservation, Body: "Described Go work.", PresentsAsAnswer: true, Citation: &interviewreport.Citation{EntryID: "entry", Revision: 2}}}},
		{name: "verdict rejected", final: true, output: []interviewreport.StatementInput{{Kind: interviewreport.KindGap, Body: "hire"}}, reason: "output_rejected"},
		{name: "unknown citation rejected", final: true, output: []interviewreport.StatementInput{{Kind: interviewreport.KindObservation, Body: "Described work.", Citation: &interviewreport.Citation{EntryID: "other", Revision: 2}}}, reason: "output_rejected"},
		{name: "wrong revision rejected", final: true, output: []interviewreport.StatementInput{{Kind: interviewreport.KindObservation, Body: "Described work.", Citation: &interviewreport.Citation{EntryID: "entry", Revision: 1}}}, reason: "output_rejected"},
		{name: "partial output is never stored", final: true, output: []interviewreport.StatementInput{{Kind: interviewreport.KindGap, Body: "Coverage unclear."}, {Kind: interviewreport.KindGap, Body: "rank"}}, reason: "output_rejected"},
		{name: "provider failure", final: true, writerErr: failure, reason: "writer_failed"},
		{name: "missing evidence waits", claimErr: interviewreportport.ErrNothingToClaim},
		{name: "duplicate delivery no writer", final: true, claimErr: interviewreportport.ErrNothingToClaim},
		{name: "claim database failure", final: true, claimErr: failure, wantErr: true},
		{name: "finish database failure", final: true, errorOnFinish: failure, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := mocks.NewInterviewReportWorkerStore(t)
			evidence := mocks.NewInterviewReportEvidence(t)
			writer := mocks.NewInterviewReportWriter(t)
			clock := mocks.NewClock(t)
			u := interviewreportusecase.Worker{Store: store, Evidence: evidence, Writer: writer, Clock: clock}
			clock.On("Now").Return(now.In(time.FixedZone("offset", 3600)).Add(123 * time.Microsecond))
			store.On("FindJob", mock.Anything, "job").Return(interviewreportport.JobRef{JobID: "job", ExecutionID: "execution"}, nil)
			entries := []interviewreportport.EvidenceEntry(nil)
			if tt.final {
				entries = []interviewreportport.EvidenceEntry{{ID: "entry", Revision: 2}}
			}
			evidence.On("Finalized", mock.Anything, "execution").Return(entries, nil)
			claim := interviewreportport.ClaimedJob{JobID: "job", GenerationID: "generation", AttemptNumber: 1, FencingToken: 2, StartedAt: now, AttemptDeadlineAt: now.Add(time.Minute), JobDeadlineAt: now.Add(5 * time.Minute)}
			store.On("ClaimAttempt", mock.Anything, "job", tt.final, now).Return(claim, tt.claimErr)
			if tt.claimErr == nil {
				writer.On("Write", mock.MatchedBy(func(ctx context.Context) bool {
					deadline, ok := ctx.Deadline()
					assert.True(t, ok)
					assert.WithinDuration(t, time.Now().Add(time.Minute), deadline, time.Second)
					return true
				}), interviewreportport.WriteRequest{ExecutionID: "execution", Evidence: entries}).Return(tt.output, tt.writerErr)
				store.On("FinishAttempt", mock.Anything, mock.MatchedBy(func(result interviewreportport.AttemptResult) bool {
					assert.Equal(t, claim, result.Claim)
					assert.Equal(t, now, result.FinishedAt)
					assert.Equal(t, tt.reason, result.FailureReason)
					if tt.reason != "" {
						assert.Empty(t, result.Statements)
					} else {
						assert.Len(t, result.Statements, len(tt.output))
					}
					return true
				})).Return(tt.errorOnFinish)
			}
			err := u.Execute(context.Background(), "job")
			if tt.wantErr {
				require.ErrorIs(t, err, failure)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// TestWorkerReconcile rechecks finalized evidence and schedules jobs without spending a provider attempt.
func TestWorkerReconcile(t *testing.T) {
	// testCase describes one independent public behavior scenario.
	type testCase struct {
		name    string
		final   bool
		wantErr bool
	}
	for _, tt := range []testCase{{name: "still waiting"}, {name: "final evidence available", final: true}} {
		t.Run(tt.name, func(t *testing.T) {
			store := mocks.NewInterviewReportWorkerStore(t)
			evidence := mocks.NewInterviewReportEvidence(t)
			writer := mocks.NewInterviewReportWriter(t)
			clock := mocks.NewClock(t)
			u := interviewreportusecase.Worker{Store: store, Evidence: evidence, Writer: writer, Clock: clock}
			now := time.Date(2026, 11, 3, 0, 0, 0, 0, time.UTC)
			clock.On("Now").Return(now)
			store.On("DueJobs", mock.Anything, now, 10).Return([]interviewreportport.JobRef{{JobID: "job", ExecutionID: "execution"}}, nil)
			entries := []interviewreportport.EvidenceEntry(nil)
			if tt.final {
				entries = []interviewreportport.EvidenceEntry{{ID: "entry", Revision: 1}}
			}
			evidence.On("Finalized", mock.Anything, "execution").Return(entries, nil)
			store.On("ReconcileJob", mock.Anything, "job", tt.final, now).Return(nil)
			err := u.Reconcile(context.Background(), 10)
			assert.Equal(t, tt.wantErr, err != nil)
			assert.NoError(t, err)
			writer.AssertNotCalled(t, "Write", mock.Anything, mock.Anything)
		})
	}
}
