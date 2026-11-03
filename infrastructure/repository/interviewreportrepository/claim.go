package interviewreportrepository

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/rfanazhari/amer-prototype/application/repository/interviewreportport"
	"github.com/rfanazhari/amer-prototype/domain/aggregate/interviewreport"
	"gorm.io/gorm"
)

// errInvalidBatch rejects an outbox claim that could not return any row.
var errInvalidBatch = errors.New("outbox claim batch size must be positive")

// Raw SQL is limited to these two claims. GORM's query helpers do not expose FOR UPDATE SKIP LOCKED
// skipping, and a lost lock here would start a second provider attempt or an uncoordinated publish.
const (
	// claimJobSQL locks the next eligible job whose window, if started, is still open.
	claimJobSQL = "SELECT id, generation_id, state, attempt_count, fencing_token, created_at, wait_deadline_at, " +
		"window_started_at, deadline_at, eligible_at, attempt_deadline_at, updated_at " +
		"FROM interview_report_jobs WHERE state IN ('pending', 'published') AND eligible_at <= ? " +
		"AND attempt_count < ? AND (deadline_at IS NULL OR deadline_at > ?) " +
		"ORDER BY eligible_at, id LIMIT 1 FOR UPDATE SKIP LOCKED"
	// claimOutboxSQL locks a batch of unconfirmed outbox rows in creation order.
	claimOutboxSQL = "SELECT id, job_kind, job_id, created_at, publish_claimed_at, confirmed_at " +
		"FROM outbox_messages WHERE confirmed_at IS NULL ORDER BY created_at, id LIMIT ? FOR UPDATE SKIP LOCKED"
)

// jobClaimColumns lists the only job columns a claim writes, so zero counts and nulls persist on purpose.
var jobClaimColumns = []string{"attempt_count", "attempt_deadline_at", "deadline_at", "fencing_token", "state", "updated_at", "window_started_at"}

// ClaimJob locks the next eligible job, inserts its next attempt and advances the attempt count and
// fencing token in one transaction. The first claim opens the job window; the attempt deadline is the
// earlier of the attempt cap and the job deadline. ErrNothingToClaim means no job was eligible.
func (s Store) ClaimJob(ctx context.Context, now time.Time) (interviewreportport.ClaimedJob, error) {
	now = now.UTC()
	var claimed interviewreportport.ClaimedJob
	err := s.inTx(ctx, func(tx *gorm.DB) error {
		var job JobModel
		found := tx.Raw(claimJobSQL, now, interviewreport.MaxAttempts, now).Scan(&job)
		if found.Error != nil {
			return found.Error
		}
		if found.RowsAffected == 0 {
			return interviewreportport.ErrNothingToClaim
		}
		windowStart, deadline := now, now.Add(interviewreport.JobWindow)
		if job.WindowStartedAt != nil && job.DeadlineAt != nil {
			windowStart, deadline = job.WindowStartedAt.UTC(), job.DeadlineAt.UTC()
		}
		attemptDeadline := now.Add(interviewreport.AttemptTimeout)
		if deadline.Before(attemptDeadline) {
			attemptDeadline = deadline
		}
		attempt := AttemptModel{
			ID: uuid.NewString(), JobID: job.ID, AttemptNumber: job.AttemptCount + 1, FencingToken: job.FencingToken + 1,
			StartedAt: now, DeadlineAt: attemptDeadline,
		}
		if err := tx.Create(&attempt).Error; err != nil {
			return err
		}
		update := tx.Model(&JobModel{}).Where("id = ?", job.ID).Select(jobClaimColumns).Updates(map[string]any{
			"attempt_count": attempt.AttemptNumber, "attempt_deadline_at": attemptDeadline, "deadline_at": deadline,
			"fencing_token": attempt.FencingToken, "state": "claimed", "updated_at": now, "window_started_at": windowStart,
		})
		if update.Error != nil {
			return update.Error
		}
		claimed = interviewreportport.ClaimedJob{
			JobID: job.ID, GenerationID: job.GenerationID, AttemptNumber: attempt.AttemptNumber, FencingToken: attempt.FencingToken,
			StartedAt: now, AttemptDeadlineAt: attemptDeadline, JobDeadlineAt: deadline,
		}
		return nil
	})
	if err != nil {
		return interviewreportport.ClaimedJob{}, err
	}
	return claimed, nil
}

// ClaimOutbox locks up to limit unconfirmed outbox rows, stamps their claim instant and returns them.
// The lock lasts only for the claim transaction, so a later pass may claim an unconfirmed row again.
// Delivery stays at-least-once; this does not confirm a publication.
func (s Store) ClaimOutbox(ctx context.Context, limit int, now time.Time) ([]interviewreportport.OutboxMessage, error) {
	if limit < 1 {
		return nil, errInvalidBatch
	}
	now = now.UTC()
	var messages []interviewreportport.OutboxMessage
	err := s.inTx(ctx, func(tx *gorm.DB) error {
		var rows []OutboxModel
		if err := tx.Raw(claimOutboxSQL, limit).Scan(&rows).Error; err != nil {
			return err
		}
		if len(rows) == 0 {
			return nil
		}
		ids := make([]string, len(rows))
		messages = make([]interviewreportport.OutboxMessage, len(rows))
		for i, row := range rows {
			ids[i] = row.ID
			messages[i] = interviewreportport.OutboxMessage{ID: row.ID, JobKind: row.JobKind, JobID: row.JobID, CreatedAt: row.CreatedAt.UTC()}
		}
		return tx.Model(&OutboxModel{}).Where("id IN ?", ids).Update("publish_claimed_at", now).Error
	})
	if err != nil {
		return nil, err
	}
	return messages, nil
}
