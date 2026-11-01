package candidateinterview

import "time"

// InterviewBudget is the platform hard-deadline budget a session must cover to start or continue.
const InterviewBudget = 15 * time.Minute

// SessionCovers reports whether the session outlives the deadline; equality is not covered.
func SessionCovers(expiresAt, deadline time.Time) bool {
	return expiresAt.After(deadline)
}
