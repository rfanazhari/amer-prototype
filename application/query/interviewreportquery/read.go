package interviewreportquery

import (
	"context"
	"errors"

	"github.com/rfanazhari/amer-prototype/application/dto"
	"github.com/rfanazhari/amer-prototype/application/repository/interviewreportport"
	"github.com/rfanazhari/amer-prototype/application/service"
	"github.com/rfanazhari/amer-prototype/domain/aggregate/interviewreport"
)

// Read returns the HR view of an interview report.
type Read struct {
	Authority   service.InterviewReportAuthority
	Assignments interviewreportport.AssignmentReader
	Reports     interviewreportport.ReportReader
}

var _ ReadOperation = Read{}

// Execute authorizes current HR before lookup and returns the newest generation with the latest succeeded account.
// A later failed or active generation never hides the previous account. It calls no provider.
func (q Read) Execute(ctx context.Context, input ReadInput) (*dto.InterviewReportView, error) {
	principal, err := q.Authority.Principal(ctx)
	if err != nil {
		return nil, err
	}
	if err := q.Authority.Authorize(ctx, principal, service.InterviewReportRead, input.AssignmentID); err != nil {
		return nil, err
	}
	if input.AssignmentID == "" {
		return nil, service.ErrReportInvalidInput
	}
	ref, err := q.Assignments.FindAssignment(ctx, input.AssignmentID)
	if err != nil {
		return nil, err
	}
	snapshot, err := q.Reports.FindByExecution(ctx, ref.ExecutionID)
	if errors.Is(err, interviewreportport.ErrNotFound) && ref.CompletedAt == nil {
		return nil, service.ErrReportNotCompleted
	}
	if err != nil {
		return nil, err
	}
	generations := snapshot.Generations
	if len(generations) == 0 {
		return nil, interviewreport.ErrInvalidReport
	}
	view := dto.InterviewReportView{
		AssignmentID: ref.AssignmentID,
		ExecutionID:  ref.ExecutionID,
		Generation:   dto.NewReportGeneration(generations[len(generations)-1]),
	}
	for i := len(generations) - 1; i >= 0; i-- {
		if generations[i].State == interviewreport.StateSucceeded {
			account := dto.NewReportAccount(generations[i])
			view.Report = &account
			break
		}
	}
	return &view, nil
}
