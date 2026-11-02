package interview

import "time"

// BeginReconnect starts a backend-detected episode under the stored episode ceiling.
func (p *Progress) BeginReconnect(now time.Time) error {
	at := trustedTime(now)
	r := &p.state.Reconnect
	if !p.activeAt(at) || r.DetectedAt != nil || r.Episodes >= 2 {
		return ErrBudget
	}
	r.Episodes++
	r.Attempts = 0
	r.DetectedAt = &at
	return nil
}

// AttemptReconnect consumes one replacement attempt strictly within 120 seconds.
func (p *Progress) AttemptReconnect(now time.Time) error {
	at := trustedTime(now)
	r := &p.state.Reconnect
	if !p.activeAt(at) || r.DetectedAt == nil || r.Attempts >= 2 || at.Before(*r.DetectedAt) || !at.Before(r.DetectedAt.Add(120*time.Second)) {
		return ErrBudget
	}
	r.Attempts++
	return nil
}

// FinishReconnect ends the episode while retaining its counters.
func (p *Progress) FinishReconnect() error {
	if !p.open() || p.state.Reconnect.DetectedAt == nil {
		return ErrConflict
	}
	p.state.Reconnect.DetectedAt = nil
	return nil
}
