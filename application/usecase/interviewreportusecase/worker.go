package interviewreportusecase

import (
	"context"
	"errors"

	"github.com/rfanazhari/amer-prototype/application/repository/interviewreportport"
	"github.com/rfanazhari/amer-prototype/application/service"
	"github.com/rfanazhari/amer-prototype/domain/aggregate/interviewreport"
)

// Worker runs report attempts without owning a broker consumer or process registration.
type Worker struct {
	Store    interviewreportport.WorkerStore
	Evidence interviewreportport.Evidence
	Writer   interviewreportport.Writer
	Clock    service.Clock
}

// Execute processes a durable job identity, propagating database errors to its caller.
func (u Worker) Execute(ctx context.Context, jobID string) error {
	ref, err := u.Store.FindJob(ctx, jobID)
	if err != nil {
		return err
	}
	evidence, err := u.Evidence.Finalized(ctx, ref.ExecutionID)
	if err != nil {
		return err
	}
	now := trustedNow(u.Clock)
	claim, err := u.Store.ClaimAttempt(ctx, jobID, len(evidence) > 0, now)
	if errors.Is(err, interviewreportport.ErrNothingToClaim) {
		return nil
	}
	if err != nil {
		return err
	}
	// The claim commit may consume time. Trusted time determines the remaining budget;
	// WithTimeout supplies the corresponding real cancellation deadline to the provider.
	remaining := claim.AttemptDeadlineAt.Sub(trustedNow(u.Clock))
	attemptCtx, cancel := context.WithTimeout(ctx, remaining)
	defer cancel()
	result := interviewreportport.AttemptResult{Claim: claim}
	if remaining > 0 {
		input := append([]interviewreportport.EvidenceEntry(nil), evidence...)
		output, writeErr := u.Writer.Write(attemptCtx, interviewreportport.WriteRequest{ExecutionID: ref.ExecutionID, Evidence: input})
		if writeErr != nil || attemptCtx.Err() != nil {
			result.FailureReason = "writer_failed"
		} else {
			result.Statements, err = validatedStatements(output, evidence)
			if err != nil {
				result.FailureReason = "output_rejected"
			}
		}
	} else {
		result.FailureReason = "attempt_expired"
	}
	result.FinishedAt = trustedNow(u.Clock)
	return u.Store.FinishAttempt(ctx, result)
}

// validatedStatements checks the entire output before any statement crosses the persistence port.
func validatedStatements(output []interviewreport.StatementInput, evidence []interviewreportport.EvidenceEntry) ([]interviewreport.Statement, error) {
	citations := make(map[interviewreport.Citation]bool, len(evidence))
	for _, entry := range evidence {
		citations[interviewreport.Citation{EntryID: entry.ID, Revision: entry.Revision}] = true
	}
	statements := make([]interviewreport.Statement, 0, len(output))
	for _, input := range output {
		statement, err := interviewreport.NewStatement(input)
		if err != nil {
			return nil, err
		}
		if citation := statement.Citation(); citation != nil && !citations[*citation] {
			return nil, interviewreport.ErrInvalidCitation
		}
		statements = append(statements, statement)
	}
	return statements, nil
}

// Reconcile processes one bounded batch of due jobs without invoking the writer.
func (u Worker) Reconcile(ctx context.Context, limit int) error {
	now := trustedNow(u.Clock)
	jobs, err := u.Store.DueJobs(ctx, now, limit)
	if err != nil {
		return err
	}
	for _, job := range jobs {
		evidence, err := u.Evidence.Finalized(ctx, job.ExecutionID)
		if err != nil {
			return err
		}
		if err := u.Store.ReconcileJob(ctx, job.JobID, len(evidence) > 0, trustedNow(u.Clock)); err != nil {
			return err
		}
	}
	return nil
}
