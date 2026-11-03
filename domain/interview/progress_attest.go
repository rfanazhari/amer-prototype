package interview

import (
	"time"

	"github.com/rfanazhari/amer-prototype/domain/enum"
)

// Attest records the model's statement that the current stage is finished.
// It needs no resolved coverage, because coverage is analysed after the interview. It seals the stage with a
// handoff to the fixed successor (none for closing) and, for a successor, makes it current in the same session.
// Authorize still requires coverage for the evidence path.
func (p *Progress) Attest(now time.Time) error {
	at := trustedTime(now)
	if !p.activeAt(at) {
		return ErrConflict
	}
	target := nextStage(p.state.CurrentStage)
	p.seal(target, at)
	if target != "" {
		p.state.CurrentStage = target
		p.state.StageActivatedAt = at
	}
	return nil
}

// AllStagesSealed reports whether every stage, including closing, has a committed handoff.
func (p *Progress) AllStagesSealed() bool {
	return p.state.CurrentStage == enum.StageClosing && len(p.state.Handoffs) == len(stages())
}
