package interviewreportrepository

import (
	"context"
	"time"

	"github.com/rfanazhari/amer-prototype/application/repository/interviewreportport"
	"gorm.io/gorm"
)

// ConfirmOutbox records broker confirmation without changing a generation or reopening a claimed job.
func (s Store) ConfirmOutbox(ctx context.Context, message interviewreportport.OutboxMessage, now time.Time) error {
	now = now.UTC().Truncate(time.Millisecond)
	return s.inTx(ctx, func(tx *gorm.DB) error {
		var row OutboxModel
		if err := tx.Where("id = ? AND job_kind = ? AND job_id = ?", message.ID, message.JobKind, message.JobID).Take(&row).Error; err != nil {
			return err
		}
		if row.ConfirmedAt != nil {
			return nil
		}
		// A consumer can claim or finish the job before the publisher records its confirm.
		// The conditional update preserves those newer states and never touches a generation.
		if err := tx.Model(&JobModel{}).Where("id = ? AND state = ?", row.JobID, "pending").
			Select("state", "updated_at").Updates(map[string]any{"state": "published", "updated_at": now}).Error; err != nil {
			return err
		}
		return tx.Model(&OutboxModel{}).Where("id = ? AND confirmed_at IS NULL", row.ID).Update("confirmed_at", now).Error
	})
}
