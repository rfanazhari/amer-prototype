package interview

import "time"

// Attest records the model's statement that the current stage is finished.
func (p *Progress) Attest(now time.Time) error { return ErrConflict }

// AllStagesSealed reports whether every stage, including closing, has a committed handoff.
func (p *Progress) AllStagesSealed() bool { return false }
