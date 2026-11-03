package interviewreportusecase

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/rfanazhari/amer-prototype/application/dto"
	"github.com/rfanazhari/amer-prototype/application/repository/interviewreportport"
	"github.com/rfanazhari/amer-prototype/application/service"
	"github.com/rfanazhari/amer-prototype/domain/aggregate/interviewreport"
)

// Admit records the first report generation for an interview completed in the caller's transaction.
type Admit struct {
	Transactions interviewreportport.Transaction
	Completion   interviewreportport.CompletionMarker
	Reports      interviewreportport.ReportReader
	Admissions   interviewreportport.AdmissionWriter
	Clock        service.Clock
	// NewID mints identities in the order report, generation, job, outbox; nil uses random UUIDs.
	NewID func() string
}

var _ AdmitOperation = Admit{}

// Execute admits the interview inside the caller's open transaction.
// It fails with ErrReportNotCompleted while the completion marker is null, so the caller rolls completion back.
// A repeated call for an execution that already has a report returns its first generation and inserts nothing.
func (u Admit) Execute(ctx context.Context, input AdmitInput) (*dto.ReportGeneration, error) {
	if input.AssignmentID == "" || input.ExecutionID == "" {
		return nil, service.ErrReportInvalidInput
	}
	var view dto.ReportGeneration
	err := u.Transactions.InTransaction(ctx, func(ctx context.Context) error {
		completedAt, err := u.Completion.CompletedAt(ctx, input.AssignmentID, input.ExecutionID)
		if err != nil {
			return err
		}
		if completedAt == nil {
			return service.ErrReportNotCompleted
		}
		existing, err := u.Reports.FindByExecution(ctx, input.ExecutionID)
		if err == nil {
			if len(existing.Generations) == 0 {
				return interviewreport.ErrInvalidReport
			}
			view = dto.NewReportGeneration(existing.Generations[0])
			return nil
		}
		if !errors.Is(err, interviewreportport.ErrNotFound) {
			return err
		}
		now := trustedNow(u.Clock)
		report, err := interviewreport.NewReport(u.mint(), input.ExecutionID, input.AssignmentID, now)
		if err != nil {
			return err
		}
		generation, err := report.StartGeneration(u.mint(), now)
		if err != nil {
			return err
		}
		admission := interviewreportport.Admission{
			Report:         report.Snapshot(),
			JobID:          u.mint(),
			OutboxID:       u.mint(),
			WaitDeadlineAt: now.Add(interviewreport.EvidenceWait),
		}
		if err := u.Admissions.Insert(ctx, admission); err != nil {
			return err
		}
		view = dto.NewReportGeneration(generation)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &view, nil
}

// mint returns the next identity from NewID, or a random UUID when none is injected.
func (u Admit) mint() string {
	if u.NewID != nil {
		return u.NewID()
	}
	return uuid.NewString()
}

// trustedNow returns the clock instant as UTC at millisecond precision.
func trustedNow(clock service.Clock) time.Time { return clock.Now().UTC().Truncate(time.Millisecond) }
