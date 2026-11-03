package interviewreportrepository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/rfanazhari/amer-prototype/application/repository/interviewreportport"
	"github.com/rfanazhari/amer-prototype/domain/aggregate/interviewreport"
	"github.com/rfanazhari/amer-prototype/infrastructure/database/transaction"
	"gorm.io/gorm"
)

// Store persists interview reports through a shared GORM database pool.
type Store struct{ DB *gorm.DB }

var (
	_ interviewreportport.Transaction      = Store{}
	_ interviewreportport.CompletionMarker = Store{}
	_ interviewreportport.ReportReader     = Store{}
	_ interviewreportport.AdmissionWriter  = Store{}
	_ interviewreportport.JobClaimer       = Store{}
	_ interviewreportport.OutboxClaimer    = Store{}
)

// completedAtSQL reads the sibling completion marker of an assignment that owns the execution pin.
const completedAtSQL = "SELECT interview_assignments.completed_at FROM interview_assignments " +
	"JOIN interview_executions ON interview_executions.assignment_id = interview_assignments.id " +
	"WHERE interview_assignments.id = ? AND interview_executions.id = ?"

// session returns the context transaction when present, otherwise the pool, bound to ctx.
func (s Store) session(ctx context.Context) *gorm.DB {
	return transaction.Session(ctx, s.DB)
}

// inTx joins the transaction carried by ctx or opens one, so callers always get all-or-nothing writes.
func (s Store) inTx(ctx context.Context, fn func(tx *gorm.DB) error) error {
	return transaction.Within(ctx, s.DB, func(_ context.Context, tx *gorm.DB) error { return fn(tx) })
}

// InTransaction runs fn in one transaction that store calls made with the handed context join.
// It joins the transaction already carried by ctx instead of opening a nested one.
func (s Store) InTransaction(ctx context.Context, fn func(ctx context.Context) error) error {
	return transaction.Within(ctx, s.DB, func(ctx context.Context, _ *gorm.DB) error { return fn(ctx) })
}

// CompletedAt returns the assignment completion instant in UTC, nil while it is null,
// or ErrNotFound when the assignment does not own the execution.
func (s Store) CompletedAt(ctx context.Context, assignmentID, executionID string) (*time.Time, error) {
	var completed sql.NullTime
	err := s.session(ctx).Raw(completedAtSQL, assignmentID, executionID).Row().Scan(&completed)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, interviewreportport.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if !completed.Valid {
		return nil, nil
	}
	utc := completed.Time.UTC()
	return &utc, nil
}

// FindByExecution loads the report, its generations and all their statements in three queries.
func (s Store) FindByExecution(ctx context.Context, executionID string) (interviewreport.Snapshot, error) {
	db := s.session(ctx)
	var report ReportModel
	if err := db.Where("execution_id = ?", executionID).Take(&report).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return interviewreport.Snapshot{}, interviewreportport.ErrNotFound
		}
		return interviewreport.Snapshot{}, err
	}
	var generations []GenerationModel
	if err := db.Where("report_id = ?", report.ID).Order("generation_number").Find(&generations).Error; err != nil {
		return interviewreport.Snapshot{}, err
	}
	var statements []StatementModel
	if len(generations) > 0 {
		ids := make([]string, len(generations))
		for i, g := range generations {
			ids[i] = g.ID
		}
		if err := db.Where("generation_id IN ?", ids).Order("generation_id, ordinal").Find(&statements).Error; err != nil {
			return interviewreport.Snapshot{}, err
		}
	}
	return toSnapshot(report, generations, statements)
}

// Insert writes the report, its first pending generation, the job and one outbox row in the caller's
// transaction, or in its own when none is open. It never touches the assignment or execution rows.
func (s Store) Insert(ctx context.Context, admission interviewreportport.Admission) error {
	gens := admission.Report.Generations
	if len(gens) != 1 || gens[0].State != interviewreport.StatePending || gens[0].Number != 1 {
		return interviewreport.ErrInvalidReport
	}
	report, generation, job, outbox := fromAdmission(admission)
	return s.inTx(ctx, func(tx *gorm.DB) error {
		for _, row := range []any{&report, &generation, &job, &outbox} {
			if err := tx.Create(row).Error; err != nil {
				return err
			}
		}
		return nil
	})
}
