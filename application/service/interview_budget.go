package service

import "time"

// InterviewBudgets carries the frozen runtime limits supplied to the lifecycle.
// These limits do not define the still-deferred post-handoff activation budget.
type InterviewBudgets struct {
	ActivationAttempts       int
	ActivationAttemptTimeout time.Duration
	ActivationWindow         time.Duration
	ReconnectWindow          time.Duration
	ReconnectAttempts        int
	ReconnectEpisodes        int
}

// DefaultInterviewBudgets returns the platform section 10 activation and recovery limits.
func DefaultInterviewBudgets() InterviewBudgets { return InterviewBudgets{} }
