package interviewreportrepository

import (
	"context"
	"errors"
	"time"

	"github.com/rfanazhari/amer-prototype/application/repository/interviewreportport"
	"github.com/rfanazhari/amer-prototype/domain/aggregate/interviewreport"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var _ interviewreportport.WorkerStore = Store{}

// workerClaimSQL locks one message's job, including states needing expiry or evidence waiting.
const workerClaimSQL = "SELECT id, generation_id, state, attempt_count, fencing_token, created_at, wait_deadline_at, " +
	"window_started_at, deadline_at, eligible_at, attempt_deadline_at, updated_at " +
	"FROM interview_report_jobs WHERE id = ? FOR UPDATE SKIP LOCKED"

// FindJob resolves the report execution for a durable job message.
func (s Store) FindJob(ctx context.Context, id string) (interviewreportport.JobRef, error) {
	var ref interviewreportport.JobRef
	err := s.session(ctx).Table("interview_report_jobs AS jobs").Select("jobs.id AS job_id, reports.execution_id").
		Joins("JOIN interview_report_generations AS generations ON generations.id = jobs.generation_id").
		Joins("JOIN interview_reports AS reports ON reports.id = generations.report_id").Where("jobs.id = ?", id).Take(&ref).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return ref, interviewreportport.ErrNotFound
	}
	return ref, err
}

// lockWorker locks generation before job so outcome and attempt transitions share one lock order.
func lockWorker(tx *gorm.DB, id string, claim bool) (GenerationModel, JobModel, error) {
	var gen GenerationModel
	subquery := tx.Model(&JobModel{}).Select("generation_id").Where("id = ?", id)
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = (?)", subquery).Take(&gen).Error; err != nil {
		return gen, JobModel{}, err
	}
	var job JobModel
	if claim {
		found := tx.Raw(workerClaimSQL, id).Scan(&job)
		if found.Error != nil {
			return gen, job, found.Error
		}
		if found.RowsAffected == 0 {
			return gen, job, interviewreportport.ErrNothingToClaim
		}
	} else if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", id).Take(&job).Error; err != nil {
		return gen, job, err
	}
	return gen, job, nil
}

// ClaimAttempt claims a message's job only when evidence and its budget permit it.
func (s Store) ClaimAttempt(ctx context.Context, id string, final bool, now time.Time) (interviewreportport.ClaimedJob, error) {
	now = now.UTC().Truncate(time.Millisecond)
	var claim interviewreportport.ClaimedJob
	err := s.inTx(ctx, func(tx *gorm.DB) error {
		gen, row, err := lockWorker(tx, id, true)
		if err != nil {
			return err
		}
		job, err := interviewreport.RestoreJob(jobSnapshot(row))
		if err != nil {
			return err
		}
		claimed := job.Claim(final, now)
		next := job.Snapshot()
		if err := s.persistBudget(tx, row, next, job.FailureReason(), now); err != nil {
			return err
		}
		if !claimed {
			return nil
		}
		if err := tx.Create(&AttemptModel{ID: mintWorkerID(), JobID: id, AttemptNumber: next.AttemptCount, FencingToken: next.FencingToken, StartedAt: now, DeadlineAt: *next.AttemptDeadlineAt}).Error; err != nil {
			return err
		}
		if gen.State == string(interviewreport.StatePending) {
			report, err := loadWorkerReport(tx, gen.ReportID)
			if err != nil {
				return err
			}
			running, err := report.Run(gen.ID, now)
			if err != nil {
				return err
			}
			if err := writeGeneration(tx, running); err != nil {
				return err
			}
		}
		claim = interviewreportport.ClaimedJob{JobID: id, GenerationID: gen.ID, AttemptNumber: next.AttemptCount, FencingToken: next.FencingToken, StartedAt: now, AttemptDeadlineAt: *next.AttemptDeadlineAt, JobDeadlineAt: *next.DeadlineAt}
		return nil
	})
	if err != nil {
		return interviewreportport.ClaimedJob{}, err
	}
	if claim.JobID == "" {
		return claim, interviewreportport.ErrNothingToClaim
	}
	return claim, nil
}

// FinishAttempt commits accepted output and outcome only for the current fenced holder.
func (s Store) FinishAttempt(ctx context.Context, result interviewreportport.AttemptResult) error {
	now := result.FinishedAt.UTC().Truncate(time.Millisecond)
	return s.inTx(ctx, func(tx *gorm.DB) error {
		gen, row, err := lockWorker(tx, result.Claim.JobID, false)
		if err != nil {
			return err
		}
		if row.GenerationID != result.Claim.GenerationID || row.AttemptCount != result.Claim.AttemptNumber {
			return nil
		}
		job, err := interviewreport.RestoreJob(jobSnapshot(row))
		if err != nil {
			return err
		}
		if !job.Finish(result.Claim.FencingToken, result.FailureReason == "", now) {
			return nil
		}
		next := job.Snapshot()
		var succeeded interviewreport.Generation
		if next.State == "succeeded" {
			report, err := loadWorkerReport(tx, gen.ReportID)
			if err != nil {
				return err
			}
			succeeded, err = report.Succeed(gen.ID, result.Statements, now)
			if err != nil {
				return err
			}
		}
		if err := s.persistBudget(tx, row, next, job.FailureReason(), now); err != nil {
			return err
		}
		if next.State != "succeeded" {
			return nil
		}
		for ordinal, statement := range succeeded.Statements {
			model := StatementModel{ID: mintWorkerID(), GenerationID: gen.ID, Ordinal: ordinal, Kind: string(statement.Kind()), Body: statement.Body(), PresentsAsAnswer: statement.PresentsAsAnswer()}
			if citation := statement.Citation(); citation != nil {
				model.EvidenceEntryID = &citation.EntryID
				model.EvidenceRevision = &citation.Revision
			}
			if err := tx.Create(&model).Error; err != nil {
				return err
			}
		}
		return writeGeneration(tx, succeeded)
	})
}

// DueJobs returns a bounded set of work needing reconciliation, including published jobs stuck in queue.
func (s Store) DueJobs(ctx context.Context, now time.Time, limit int) ([]interviewreportport.JobRef, error) {
	if limit < 1 {
		return nil, errInvalidBatch
	}
	var refs []interviewreportport.JobRef
	now = now.UTC().Truncate(time.Millisecond)
	err := s.session(ctx).Table("interview_report_jobs AS jobs").Select("jobs.id AS job_id, reports.execution_id").
		Joins("JOIN interview_report_generations AS generations ON generations.id = jobs.generation_id").
		Joins("JOIN interview_reports AS reports ON reports.id = generations.report_id").
		Where("(jobs.state IN ? AND jobs.eligible_at <= ?) OR (jobs.state = ? AND jobs.attempt_deadline_at <= ?) OR (jobs.state IN ? AND (jobs.deadline_at <= ? OR (jobs.deadline_at IS NULL AND jobs.wait_deadline_at <= ?)))", []string{"pending", "waiting"}, now, "claimed", now, []string{"pending", "published", "waiting"}, now, now).
		Order("jobs.eligible_at, jobs.id").Limit(limit).Scan(&refs).Error
	return refs, err
}

// ReconcileJob recovers abandoned attempts and records durable retry or evidence publication.
func (s Store) ReconcileJob(ctx context.Context, id string, final bool, now time.Time) error {
	now = now.UTC().Truncate(time.Millisecond)
	return s.inTx(ctx, func(tx *gorm.DB) error {
		_, row, err := lockWorker(tx, id, false)
		if err != nil {
			return err
		}
		job, err := interviewreport.RestoreJob(jobSnapshot(row))
		if err != nil {
			return err
		}
		publish := job.Reconcile(final, now)
		if err := s.persistBudget(tx, row, job.Snapshot(), job.FailureReason(), now); err != nil {
			return err
		}
		if !publish {
			return nil
		}
		return tx.Create(&OutboxModel{ID: mintWorkerID(), JobKind: jobKind, JobID: id, CreatedAt: now}).Error
	})
}
