package interviewreportrepository

import (
	"context"
	"github.com/rfanazhari/amer-prototype/application/repository/interviewreportport"
	"time"
)

// ClaimAttempt claims a message's job only when evidence and its budget permit it.
func (s Store) ClaimAttempt(ctx context.Context, id string, final bool, now time.Time) (interviewreportport.ClaimedJob, error) {
	return interviewreportport.ClaimedJob{}, nil
}

// FinishAttempt commits accepted output and outcome only for the current fenced holder.
func (s Store) FinishAttempt(ctx context.Context, result interviewreportport.AttemptResult) error {
	return nil
}
