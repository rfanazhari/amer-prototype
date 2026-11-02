package interviewusecase

import (
	"context"

	"github.com/rfanazhari/amer-prototype/application/repository/interviewport"
	"github.com/rfanazhari/amer-prototype/domain/enum"
	"github.com/rfanazhari/amer-prototype/domain/interview"
)

// Propose commits a handoff and closes the source shell without opening its successor.
// Authorization is repeated inside the locked callback; narrative failure is nonfatal.
func (u Workflow) Propose(ctx context.Context, req Proposal) error {
	record, err := u.findCurrent(ctx, req.Authority)
	if err != nil {
		return err
	}
	if record.Version != req.ProgressVersion {
		return interviewport.ErrConflict
	}
	now := u.now()
	p, err := interview.RestoreProgress(record.Snapshot)
	if err != nil {
		return err
	}
	if err = p.CommitHandoff(req.Target, now); err != nil {
		return err
	}
	h := p.Snapshot().Handoffs[len(p.Snapshot().Handoffs)-1]
	var summary string
	if u.Summary != nil {
		text, summaryErr := u.Summary.Summarize(ctx, h)
		if summaryErr == nil {
			summary = text
		}
	}
	return u.update(ctx, req.Authority, req.ProgressVersion, func(r *interviewport.ProgressRecord) error {
		committedAt := u.now()
		agg, err := interview.RestoreProgress(r.Snapshot)
		if err != nil {
			return err
		}
		if err = agg.CommitHandoff(req.Target, committedAt); err != nil {
			return err
		}
		r.Snapshot = agg.Snapshot()
		if summary != "" {
			if r.HandoffSummaries == nil {
				r.HandoffSummaries = make(map[enum.QuestionStage]string)
			}
			r.HandoffSummaries[h.Stage] = summary
		}
		closeSessions(r, committedAt)
		return nil
	})
}
