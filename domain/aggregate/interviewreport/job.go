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

// copyTime detaches and normalizes an optional durable instant.
func copyTime(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}
	result := value.UTC().Truncate(time.Millisecond)
	return &result
}

// detach copies all optional instants so callers cannot alter a restored budget.
func (s JobSnapshot) detach() JobSnapshot {
	s.WindowStartedAt = copyTime(s.WindowStartedAt)
	s.DeadlineAt = copyTime(s.DeadlineAt)
	s.AttemptDeadlineAt = copyTime(s.AttemptDeadlineAt)
	s.CreatedAt = s.CreatedAt.UTC().Truncate(time.Millisecond)
	s.WaitDeadlineAt = s.WaitDeadlineAt.UTC().Truncate(time.Millisecond)
	s.EligibleAt = s.EligibleAt.UTC().Truncate(time.Millisecond)
	s.UpdatedAt = s.UpdatedAt.UTC().Truncate(time.Millisecond)
	return s
}

// RestoreJob restores the durable attempt budget without invoking a provider.
func RestoreJob(s JobSnapshot) (*Job, error) {
	if s.ID == "" || s.GenerationID == "" || s.CreatedAt.IsZero() || s.UpdatedAt.Before(s.CreatedAt) || s.AttemptCount < 0 || s.AttemptCount > MaxAttempts || s.FencingToken < 1 || !s.WaitDeadlineAt.Equal(s.CreatedAt.Add(EvidenceWait)) {
		return nil, ErrInvalidReport
	}
	switch s.State {
	case "pending", "published", "waiting", "claimed", "succeeded", "failed", "expired":
	default:
		return nil, ErrInvalidReport
	}
	if (s.WindowStartedAt == nil) != (s.DeadlineAt == nil) {
		return nil, ErrInvalidReport
	}
	if s.WindowStartedAt != nil && !s.DeadlineAt.Equal(s.WindowStartedAt.Add(JobWindow)) {
		return nil, ErrInvalidReport
	}
	if s.State == "claimed" && (s.AttemptCount == 0 || s.AttemptDeadlineAt == nil || s.DeadlineAt == nil) {
		return nil, ErrInvalidReport
	}
	return &Job{state: s.detach()}, nil
}

// Snapshot returns detached budget state.
func (j *Job) Snapshot() JobSnapshot { return j.state.detach() }

// terminal reports that the durable job must never call a provider again.
func (j *Job) terminal() bool {
	return j.state.State == "succeeded" || j.state.State == "failed" || j.state.State == "expired"
}

// ready applies expiry and evidence rules before a claim or reconciler publication.
func (j *Job) ready(final bool, now time.Time) bool {
	s := &j.state
	if j.terminal() || now.Before(s.UpdatedAt) {
		return false
	}
	if s.State == "claimed" {
		if !now.Before(*s.AttemptDeadlineAt) {
			j.Finish(s.FencingToken, false, now)
		}
		return false
	}
	if (s.DeadlineAt != nil && !now.Before(*s.DeadlineAt)) || (s.DeadlineAt == nil && !final && !now.Before(s.WaitDeadlineAt)) {
		s.State = "expired"
		s.UpdatedAt = now
		return false
	}
	if s.AttemptCount >= MaxAttempts {
		s.State = "failed"
		s.UpdatedAt = now
		return false
	}
	if now.Before(s.EligibleAt) {
		return false
	}
	if !final {
		// Evidence that disappears after a first attempt cannot reopen or reset its window.
		if s.WindowStartedAt == nil {
			s.State = "waiting"
		}
		s.EligibleAt = now.Add(EvidenceRecheck)
		s.UpdatedAt = now
		return false
	}
	return true
}

// Claim starts an eligible attempt only after final evidence exists.
func (j *Job) Claim(final bool, now time.Time) bool {
	now = now.UTC().Truncate(time.Millisecond)
	if !j.ready(final, now) {
		return false
	}
	s := &j.state
	if s.WindowStartedAt == nil {
		start, end := now, now.Add(JobWindow)
		s.WindowStartedAt = &start
		s.DeadlineAt = &end
	}
	deadline := now.Add(AttemptTimeout)
	if s.DeadlineAt.Before(deadline) {
		deadline = *s.DeadlineAt
	}
	s.AttemptDeadlineAt = &deadline
	s.AttemptCount++
	s.FencingToken++
	s.State = "claimed"
	s.UpdatedAt = now
	return true
}

// Finish fences the holder and records success or schedules a bounded retry.
func (j *Job) Finish(token int64, accepted bool, now time.Time) bool {
	now = now.UTC().Truncate(time.Millisecond)
	s := &j.state
	if s.State != "claimed" || token != s.FencingToken || now.Before(s.UpdatedAt) {
		return false
	}
	accepted = accepted && now.Before(*s.AttemptDeadlineAt) && now.Before(*s.DeadlineAt)
	s.FencingToken++
	s.UpdatedAt = now
	s.AttemptDeadlineAt = nil
	switch {
	case accepted:
		s.State = "succeeded"
	case !now.Before(*s.DeadlineAt):
		s.State = "expired"
	case s.AttemptCount >= MaxAttempts:
		s.State = "failed"
	default:
		s.EligibleAt = now.Add(RetryDelay)
		if !s.EligibleAt.Before(*s.DeadlineAt) {
			s.State = "expired"
		} else {
			s.State = "pending"
		}
	}
	return true
}

// Reconcile expires abandoned attempts and republishes eligible work without claiming it.
func (j *Job) Reconcile(final bool, now time.Time) bool {
	now = now.UTC().Truncate(time.Millisecond)
	if !j.ready(final, now) {
		return false
	}
	if j.state.State == "published" {
		return false
	}
	j.state.State = "published"
	j.state.UpdatedAt = now
	return true
}

// FailureReason returns the safe terminal generation failure, empty for active or succeeded jobs.
func (j *Job) FailureReason() string {
	switch j.state.State {
	case "expired":
		if j.state.WindowStartedAt == nil {
			return "evidence_unavailable"
		}
		return "budget_expired"
	case "failed":
		return "attempts_exhausted"
	default:
		return ""
	}
}
