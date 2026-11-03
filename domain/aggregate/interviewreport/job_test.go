package interviewreport_test

import (
	"github.com/rfanazhari/amer-prototype/domain/aggregate/interviewreport"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

// jobEpoch is the deterministic admission instant for budget scenarios.
var jobEpoch = time.Date(2026, 11, 3, 0, 0, 0, 0, time.UTC)

// budgetJob restores a fresh admitted budget with an independent snapshot.
func budgetJob(t *testing.T) *interviewreport.Job {
	t.Helper()
	j, err := interviewreport.RestoreJob(interviewreport.JobSnapshot{ID: "job", GenerationID: "generation", State: "published", FencingToken: 1, CreatedAt: jobEpoch, WaitDeadlineAt: jobEpoch.Add(30 * time.Minute), EligibleAt: jobEpoch, UpdatedAt: jobEpoch})
	require.NoError(t, err)
	return j
}

// TestJobBudget proves accepted attempt limits, exact boundaries, fencing and asynchronous evidence timing.
func TestJobBudget(t *testing.T) {
	type testCase struct {
		name string
		run  func(*testing.T, *interviewreport.Job)
	}
	tests := []testCase{
		{name: "first evidence claim starts sixty seconds and five minutes after queued admission", run: func(t *testing.T, j *interviewreport.Job) {
			now := jobEpoch.Add(20 * time.Minute)
			require.True(t, j.Claim(true, now))
			s := j.Snapshot()
			assert.Equal(t, 1, s.AttemptCount)
			assert.Equal(t, now.Add(time.Minute), *s.AttemptDeadlineAt)
			assert.Equal(t, now.Add(5*time.Minute), *s.DeadlineAt)
			assert.False(t, j.Claim(true, now.Add(time.Second)), "duplicate must not claim")
		}},
		{name: "five second retry delay and no third attempt", run: func(t *testing.T, j *interviewreport.Job) {
			require.True(t, j.Claim(true, jobEpoch))
			require.True(t, j.Finish(j.Snapshot().FencingToken, false, jobEpoch.Add(time.Second)))
			assert.False(t, j.Claim(true, jobEpoch.Add(6*time.Second-time.Millisecond)))
			require.True(t, j.Claim(true, jobEpoch.Add(6*time.Second)))
			require.True(t, j.Finish(j.Snapshot().FencingToken, false, jobEpoch.Add(7*time.Second)))
			assert.Equal(t, "failed", j.Snapshot().State)
			assert.False(t, j.Claim(true, jobEpoch.Add(12*time.Second)))
			assert.Equal(t, 2, j.Snapshot().AttemptCount)
		}},
		{name: "queue time after first claim consumes window and caps next attempt", run: func(t *testing.T, j *interviewreport.Job) {
			require.True(t, j.Claim(true, jobEpoch))
			require.True(t, j.Finish(j.Snapshot().FencingToken, false, jobEpoch.Add(time.Second)))
			require.True(t, j.Claim(true, jobEpoch.Add(4*time.Minute+50*time.Second)))
			assert.Equal(t, jobEpoch.Add(5*time.Minute), *j.Snapshot().AttemptDeadlineAt)
		}},
		{name: "queue expiry at exact five minute boundary claims no attempt", run: func(t *testing.T, j *interviewreport.Job) {
			require.True(t, j.Claim(true, jobEpoch))
			require.True(t, j.Finish(j.Snapshot().FencingToken, false, jobEpoch.Add(time.Second)))
			assert.False(t, j.Claim(true, jobEpoch.Add(5*time.Minute)))
			assert.Equal(t, "expired", j.Snapshot().State)
			assert.Equal(t, 1, j.Snapshot().AttemptCount)
		}},
		{name: "waiting consumes no attempt and starts no window", run: func(t *testing.T, j *interviewreport.Job) {
			assert.False(t, j.Claim(false, jobEpoch))
			s := j.Snapshot()
			assert.Equal(t, "waiting", s.State)
			assert.Zero(t, s.AttemptCount)
			assert.Nil(t, s.WindowStartedAt)
			assert.Nil(t, s.DeadlineAt)
			assert.Equal(t, jobEpoch.Add(30*time.Second), s.EligibleAt)
			assert.False(t, j.Reconcile(false, jobEpoch.Add(30*time.Second)))
			assert.True(t, j.Reconcile(true, jobEpoch.Add(time.Minute)))
			assert.Equal(t, "published", j.Snapshot().State)
			require.True(t, j.Claim(true, jobEpoch.Add(time.Minute)))
			assert.Equal(t, jobEpoch.Add(6*time.Minute), *j.Snapshot().DeadlineAt)
		}},
		{name: "evidence give up at thirty minutes even on late final arrival", run: func(t *testing.T, j *interviewreport.Job) {
			assert.False(t, j.Claim(false, jobEpoch))
			assert.False(t, j.Claim(true, jobEpoch.Add(30*time.Minute)))
			assert.Equal(t, "expired", j.Snapshot().State)
			assert.Equal(t, "evidence_unavailable", j.FailureReason())
			assert.Zero(t, j.Snapshot().AttemptCount)
		}},
		{name: "attempt deadline equality rejects successful late output and fences old holder", run: func(t *testing.T, j *interviewreport.Job) {
			require.True(t, j.Claim(true, jobEpoch))
			token := j.Snapshot().FencingToken
			require.True(t, j.Finish(token, true, jobEpoch.Add(time.Minute)))
			assert.Equal(t, "pending", j.Snapshot().State)
			assert.False(t, j.Finish(token, true, jobEpoch.Add(time.Minute)))
			assert.Equal(t, jobEpoch.Add(65*time.Second), j.Snapshot().EligibleAt)
		}},
		{name: "reconcile fences abandoned attempt and republishes after delay", run: func(t *testing.T, j *interviewreport.Job) {
			require.True(t, j.Claim(true, jobEpoch))
			token := j.Snapshot().FencingToken
			assert.False(t, j.Reconcile(true, jobEpoch.Add(time.Minute)))
			assert.False(t, j.Finish(token, true, jobEpoch.Add(time.Minute)))
			assert.True(t, j.Reconcile(true, jobEpoch.Add(65*time.Second)))
			assert.False(t, j.Reconcile(true, jobEpoch.Add(65*time.Second)), "no duplicate publication")
			require.True(t, j.Claim(true, jobEpoch.Add(65*time.Second)))
			require.True(t, j.Finish(j.Snapshot().FencingToken, true, jobEpoch.Add(66*time.Second)))
			assert.Equal(t, "succeeded", j.Snapshot().State)
		}},
		{name: "snapshots own timestamp pointers", run: func(t *testing.T, j *interviewreport.Job) {
			require.True(t, j.Claim(true, jobEpoch))
			s := j.Snapshot()
			*s.DeadlineAt = jobEpoch
			assert.Equal(t, jobEpoch.Add(5*time.Minute), *j.Snapshot().DeadlineAt)
			restored, err := interviewreport.RestoreJob(j.Snapshot())
			require.NoError(t, err)
			s = restored.Snapshot()
			*s.AttemptDeadlineAt = jobEpoch
			assert.Equal(t, jobEpoch.Add(time.Minute), *restored.Snapshot().AttemptDeadlineAt)
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) { tt.run(t, budgetJob(t)) })
	}
}
