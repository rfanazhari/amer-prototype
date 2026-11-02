package interviewusecase

import (
	"context"

	"github.com/rfanazhari/amer-prototype/application/service"
	"github.com/rfanazhari/amer-prototype/domain/interview"
)

// failReasonDeadline is the lifecycle failure reason when the hard deadline passes without the required record.
const failReasonDeadline = "interview_required_record_missing_at_deadline"

// FinishRequest carries the lifecycle authority for one terminal decision.
type FinishRequest struct {
	Authority service.InterviewAttempt
}

// Finish decides one terminal lifecycle outcome from the committed record and trusted time.
// The required record completes; its absence at the hard deadline fails. Neither path writes
// interview rows, creates a report or outbox row, or removes earlier handoffs, so a lifecycle
// rejection leaves the committed record intact. Before the deadline an incomplete record is a conflict.
func (u Workflow) Finish(ctx context.Context, req FinishRequest) error {
	record, err := u.findCurrent(ctx, req.Authority)
	if err != nil {
		return err
	}
	p, err := interview.RestoreProgress(record.Snapshot)
	if err != nil {
		return err
	}
	switch {
	case p.ReadyForCompletion():
		return u.Lifecycle.Complete(ctx, service.InterviewCompletion{Authority: req.Authority, ProgressReference: req.Authority.ExecutionID})
	case p.HardDeadlineReached(u.now()):
		return u.Lifecycle.Fail(ctx, service.InterviewFailure{Authority: req.Authority, Reason: failReasonDeadline})
	default:
		return interview.ErrConflict
	}
}
