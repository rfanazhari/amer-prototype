package execution

import "time"

// RecordAttempt advances authority on a running execution without changing its activation instant.
func (e *Execution) RecordAttempt(expectedVersion, generation int64, now time.Time) (Snapshot, error) {
 if e.state.State!=StateRunning{return Snapshot{},ErrInvalidTransition}
 if e.state.StateVersion!=expectedVersion{return Snapshot{},ErrStaleVersion}
 if generation<=e.state.CurrentGeneration{return Snapshot{},ErrStaleAttempt}
 e.state.CurrentGeneration=generation
 e.state.StateVersion++
 e.state.UpdatedAt=now
 return e.state.detach(),nil
}
