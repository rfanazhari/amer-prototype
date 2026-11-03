package interview

import (
	"time"

	"github.com/rfanazhari/amer-prototype/domain/enum"
)

// nextStage returns the fixed successor; closing has no next stage.
func nextStage(stage enum.QuestionStage) enum.QuestionStage {
	index := stageIndex(stage)
	if index < 0 || index == 3 {
		return ""
	}
	return stages()[index+1]
}

// stageCovered requires terminal coverage and no unresolved recording gap in a stage.
func (p *Progress) stageCovered(stage enum.QuestionStage) bool {
	for _, q := range p.state.Coverage {
		if q.Stage == stage && ((q.State != Answered && q.State != Omitted) || p.unresolved(q.QuestionID)) {
			return false
		}
	}
	return true
}

// Authorize checks a fixed next-stage proposal without changing progress.
// Coverage must be resolved explicitly, including when the soft budget has elapsed.
func (p *Progress) Authorize(target enum.QuestionStage, now time.Time) error {
	if !p.activeAt(trustedTime(now)) || target != nextStage(p.state.CurrentStage) || !p.stageCovered(p.state.CurrentStage) {
		return ErrConflict
	}
	return nil
}

// CommitHandoff seals the source's structured evidence after authorization.
// Persistence must commit this aggregate before the caller opens a target session.
func (p *Progress) CommitHandoff(target enum.QuestionStage, now time.Time) error {
	if err := p.Authorize(target, now); err != nil {
		return err
	}
	p.seal(target, trustedTime(now))
	return nil
}

// seal appends the current stage's handoff with its coverage, final entry ids and technical losses as stored.
func (p *Progress) seal(target enum.QuestionStage, at time.Time) {
	h := Handoff{Stage: p.state.CurrentStage, Target: target, CommittedAt: at}
	for _, q := range p.state.Coverage {
		if q.Stage == h.Stage {
			h.Coverage = append(h.Coverage, q)
		}
	}
	for _, entry := range p.state.Entries {
		if entry.Stage == h.Stage && entry.State == Final {
			h.EntryIDs = append(h.EntryIDs, entry.ID)
		}
	}
	for _, loss := range p.state.Losses {
		for _, q := range h.Coverage {
			if q.QuestionID == loss.QuestionID {
				h.Losses = append(h.Losses, loss)
			}
		}
	}
	p.state.Handoffs = append(p.state.Handoffs, h)
}

// ActivateStage requires a committed source handoff before starting the target timer.
func (p *Progress) ActivateStage(target enum.QuestionStage, now time.Time) error {
	at := trustedTime(now)
	if p.open() || p.state.ExecutionID == "" || target == "" || target != nextStage(p.state.CurrentStage) || p.HardDeadlineReached(at) {
		return ErrConflict
	}
	index := stageIndex(p.state.CurrentStage)
	if len(p.state.Handoffs) != index+1 {
		return ErrConflict
	}
	h := p.state.Handoffs[index]
	if h.Target != target || at.Before(h.CommittedAt) {
		return ErrConflict
	}
	p.state.CurrentStage = target
	p.state.StageActivatedAt = at
	return nil
}

// SoftBudgetReached includes equality at the published stage deadline.
func (p *Progress) SoftBudgetReached(now time.Time) bool {
	seconds := stageSeconds(p.state.Configuration, p.state.CurrentStage)
	return seconds > 0 && !p.state.StageActivatedAt.IsZero() && !trustedTime(now).Before(p.state.StageActivatedAt.Add(time.Duration(seconds)*time.Second))
}

// HardDeadlineReached includes equality at fifteen minutes from running.
func (p *Progress) HardDeadlineReached(now time.Time) bool {
	return !p.state.RunningAt.IsZero() && !trustedTime(now).Before(p.state.RunningAt.Add(15*time.Minute))
}

// ReadyForCompletion requires all coverage, resolved gaps and four sealed handoffs.
// The coordinator establishes durable commit and asks the lifecycle to complete.
func (p *Progress) ReadyForCompletion() bool {
	if p.state.CurrentStage != enum.StageClosing || len(p.state.Handoffs) != 4 {
		return false
	}
	for _, stage := range stages() {
		if !p.stageCovered(stage) {
			return false
		}
	}
	return true
}
