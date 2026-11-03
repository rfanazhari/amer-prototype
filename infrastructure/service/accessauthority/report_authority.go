package accessauthority

import (
	"context"

	"github.com/rfanazhari/amer-prototype/application/service"
)

// ReportAuthority is the interview report view of Authority. It shares the session resolver and clock
// of the converted Authority and exists only because one Go type cannot carry two Authorize methods.
type ReportAuthority Authority

var _ service.InterviewReportAuthority = ReportAuthority{}

// Principal returns the stable account UUID after current session and account binding succeed.
func (a ReportAuthority) Principal(ctx context.Context) (service.AdminPrincipal, error) {
	return Authority(a).Principal(ctx)
}

// Authorize rechecks the bound session and permits only current HR report read and regenerate operations.
func (a ReportAuthority) Authorize(ctx context.Context, principal service.AdminPrincipal, operation, assignmentID string) error {
	_ = assignmentID
	return Authority(a).authorize(ctx, principal, service.KnownInterviewReportOperation(operation), service.AllowsInterviewReport)
}
