package interviewreportquery

import (
	"context"
	"errors"

	"github.com/rfanazhari/amer-prototype/application/dto"
	"github.com/rfanazhari/amer-prototype/application/repository/interviewreportport"
	"github.com/rfanazhari/amer-prototype/application/service"
)

// Read returns the HR view of an interview report.
type Read struct {
	Authority   service.InterviewReportAuthority
	Assignments interviewreportport.AssignmentReader
	Reports     interviewreportport.ReportReader
}

var _ ReadOperation = Read{}

// errNotImplemented marks the behavior that the GREEN commit supplies.
var errNotImplemented = errors.New("not implemented")

// Execute authorizes current HR before lookup and returns the newest generation with the latest succeeded account.
func (q Read) Execute(ctx context.Context, input ReadInput) (*dto.InterviewReportView, error) {
	return nil, errNotImplemented
}
