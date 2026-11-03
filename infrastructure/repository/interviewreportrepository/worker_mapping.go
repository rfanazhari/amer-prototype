package interviewreportrepository

import (
	"reflect"
	"time"

	"github.com/google/uuid"
	"github.com/rfanazhari/amer-prototype/domain/aggregate/interviewreport"
	"gorm.io/gorm"
)

// mintWorkerID mints a durable attempt, statement or outbox identity.
func mintWorkerID() string { return uuid.NewString() }

// jobSnapshot maps adapter state into the domain budget, which detaches optional times on restoration.
func jobSnapshot(row JobModel) interviewreport.JobSnapshot {
	return interviewreport.JobSnapshot{ID: row.ID, GenerationID: row.GenerationID, State: row.State, AttemptCount: row.AttemptCount, FencingToken: row.FencingToken, CreatedAt: row.CreatedAt.UTC().Truncate(time.Millisecond), WaitDeadlineAt: row.WaitDeadlineAt.UTC().Truncate(time.Millisecond), EligibleAt: row.EligibleAt.UTC().Truncate(time.Millisecond), UpdatedAt: row.UpdatedAt.UTC().Truncate(time.Millisecond), WindowStartedAt: workerTime(row.WindowStartedAt), DeadlineAt: workerTime(row.DeadlineAt), AttemptDeadlineAt: workerTime(row.AttemptDeadlineAt)}
}

// workerTime detaches a database timestamp and normalizes it to trusted application precision.
func workerTime(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}
	result := value.UTC().Truncate(time.Millisecond)
	return &result
}

// loadWorkerReport restores generation history once so lifecycle outcomes remain aggregate-owned.
func loadWorkerReport(tx *gorm.DB, id string) (*interviewreport.Report, error) {
	var row ReportModel
	if err := tx.Where("id = ?", id).Take(&row).Error; err != nil {
		return nil, err
	}
	var gens []GenerationModel
	if err := tx.Where("report_id = ?", id).Order("generation_number").Find(&gens).Error; err != nil {
		return nil, err
	}
	ids := make([]string, len(gens))
	for i, gen := range gens {
		ids[i] = gen.ID
	}
	var statements []StatementModel
	if err := tx.Where("generation_id IN ?", ids).Order("generation_id, ordinal").Find(&statements).Error; err != nil {
		return nil, err
	}
	snapshot, err := toSnapshot(row, gens, statements)
	if err != nil {
		return nil, err
	}
	return interviewreport.RestoreReport(snapshot)
}

// writeGeneration explicitly writes null reasons and the aggregate's new lifecycle state.
func writeGeneration(tx *gorm.DB, g interviewreport.Generation) error {
	var reason *string
	if g.FailureReason != "" {
		value := g.FailureReason
		reason = &value
	}
	return tx.Model(&GenerationModel{}).Where("id = ?", g.ID).Select("failure_reason", "state", "updated_at").Updates(map[string]any{"failure_reason": reason, "state": string(g.State), "updated_at": g.UpdatedAt}).Error
}

// persistBudget stores budget changes, closes any fenced attempt and applies terminal failure atomically.
func (s Store) persistBudget(tx *gorm.DB, old JobModel, next interviewreport.JobSnapshot, reason string, now time.Time) error {
	if reflect.DeepEqual(jobSnapshot(old), next) {
		return nil
	}
	if old.State == "claimed" && next.FencingToken > old.FencingToken {
		outcome := "failed"
		if next.State == "succeeded" {
			outcome = "succeeded"
		} else if old.AttemptDeadlineAt != nil && !now.Before(*old.AttemptDeadlineAt) {
			outcome = "expired"
		}
		result := tx.Model(&AttemptModel{}).Where("job_id = ? AND attempt_number = ? AND fencing_token = ? AND finished_at IS NULL", old.ID, old.AttemptCount, old.FencingToken).
			Select("finished_at", "outcome").Updates(map[string]any{"finished_at": now, "outcome": outcome})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return interviewreport.ErrInvalidTransition
		}
	}
	values := map[string]any{"attempt_count": next.AttemptCount, "attempt_deadline_at": next.AttemptDeadlineAt, "deadline_at": next.DeadlineAt, "eligible_at": next.EligibleAt, "fencing_token": next.FencingToken, "state": next.State, "updated_at": next.UpdatedAt, "window_started_at": next.WindowStartedAt}
	if err := tx.Model(&JobModel{}).Where("id = ?", old.ID).Select("attempt_count", "attempt_deadline_at", "deadline_at", "eligible_at", "fencing_token", "state", "updated_at", "window_started_at").Updates(values).Error; err != nil {
		return err
	}
	if reason == "" {
		return nil
	}
	var gen GenerationModel
	if err := tx.Where("id = ?", old.GenerationID).Take(&gen).Error; err != nil {
		return err
	}
	report, err := loadWorkerReport(tx, gen.ReportID)
	if err != nil {
		return err
	}
	failed, err := report.Fail(gen.ID, reason, now)
	if err != nil {
		return err
	}
	return writeGeneration(tx, failed)
}
