package accessauthority

import (
	"context"

	"github.com/rfanazhari/amer-prototype/application/service"
)

// ReportAuthority is the interview report view of Authority. It shares the session resolver and clock
// of the converted Authority and exists only because one Go type cannot carry two Authorize methods.
type ReportAuthority Authority

var _ service.InterviewReportAuthority = ReportAuthority{}

// Principal is not implemented yet.
func (a ReportAuthority) Principal(ctx context.Context) (service.AdminPrincipal, error) {
	return service.AdminPrincipal{}, nil
}

// Authorize is not implemented yet.
func (a ReportAuthority) Authorize(ctx context.Context, principal service.AdminPrincipal, operation, assignmentID string) error {
	return nil
}
