package execution

import "time"

// RecordAttempt advances authority on a running execution without changing its activation instant.
func (e *Execution) RecordAttempt(_ int64, _ int64, _ time.Time) (Snapshot, error) {
	return Snapshot{}, ErrInvalidTransition
}
