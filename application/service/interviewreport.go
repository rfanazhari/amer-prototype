package service

import "context"

// InterviewReportAuthority separates trusted session resolution from report authorization.
// It is distinct from AgentConfigurationAuthority so a report operation never passes through the
// configuration operation catalog. One adapter may implement both on a shared session lookup.
type InterviewReportAuthority interface {
	// Principal resolves the authenticated administrative session from trusted context and
	// rejects candidate sessions, expired or revoked sessions and missing credentials.
	Principal(context.Context) (AdminPrincipal, error)
	// Authorize rechecks current authority for a report operation (InterviewReportRead or
	// InterviewReportRegenerate) on an assignment before any lookup or disclosure.
	Authorize(ctx context.Context, principal AdminPrincipal, operation, assignmentID string) error
}
