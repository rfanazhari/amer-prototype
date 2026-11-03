package interviewreport

import "time"

const (
	// MaxAttempts is the total provider attempts allowed inside one report job.
	MaxAttempts = 2
	// AttemptTimeout is the longest a single provider attempt may run.
	AttemptTimeout = 60 * time.Second
	// RetryDelay is the wait after a failed attempt before the next one becomes eligible.
	RetryDelay = 5 * time.Second
	// JobWindow is the time allowed from the first claim with final evidence to job expiry.
	JobWindow = 5 * time.Minute
	// EvidenceWait is the time after admission before a job still waiting for final evidence gives up.
	EvidenceWait = 30 * time.Minute
	// EvidenceRecheck is the interval at which a waiting job is checked for final evidence.
	EvidenceRecheck = 30 * time.Second
)
