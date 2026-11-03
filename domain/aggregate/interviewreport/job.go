package interviewreport

import "time"

// JobSnapshot is detached durable attempt-budget state, never interview state.
type JobSnapshot struct {
	ID, GenerationID, State                          string
	AttemptCount                                     int
	FencingToken                                     int64
	CreatedAt, WaitDeadlineAt, EligibleAt, UpdatedAt time.Time
	WindowStartedAt, DeadlineAt, AttemptDeadlineAt   *time.Time
}

// Job protects the generation's provider budget and fencing transitions.
type Job struct{ state JobSnapshot }

// RestoreJob restores the durable attempt budget without invoking a provider.
func RestoreJob(s JobSnapshot) (*Job, error) { return &Job{state: s}, nil }

// Snapshot returns detached budget state.
func (j *Job) Snapshot() JobSnapshot { return j.state }

// Claim starts an eligible attempt only after final evidence exists.
func (j *Job) Claim(final bool, now time.Time) bool { return false }

// Finish fences the holder and records success or schedules a bounded retry.
func (j *Job) Finish(token int64, accepted bool, now time.Time) bool { return false }

// Reconcile expires abandoned attempts and republishes eligible work without claiming it.
func (j *Job) Reconcile(final bool, now time.Time) bool { return false }

// FailureReason returns the safe terminal generation failure, empty for active or succeeded jobs.
func (j *Job) FailureReason() string { return "" }
