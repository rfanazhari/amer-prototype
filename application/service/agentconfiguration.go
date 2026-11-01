package service

import "context"

// AdminPrincipal is an authenticated subject resolved by the trusted access
// adapter, never decoded from request JSON, a caller role, or arbitrary headers.
// It grants no permissions on its own; Authorize must consult current authority.
type AdminPrincipal struct {
	SubjectID string
}

// AgentConfigurationAuthority separates trusted session resolution from scoped
// authorization. No real authentication adapter or permissive fallback is supplied.
type AgentConfigurationAuthority interface {
	// Principal resolves the authenticated admin session from trusted context and
	// rejects candidate sessions, expired/revoked sessions and missing credentials.
	Principal(context.Context) (AdminPrincipal, error)
	// Authorize checks current authority for an operation and owned resource before
	// any read, mutation or replay disclosure. An empty resource denotes collection
	// access; operation names are internal, not caller-supplied roles. Session/CSRF
	// verification remains the real access adapter's responsibility before exposure.
	Authorize(context.Context, AdminPrincipal, string, string) error
}

// AgentReference identifies an immutable reference version, not a latest alias.
// Its concrete kinds and accepted identity formats remain T1/T5 decisions.
type AgentReference struct {
	Kind    string
	ID      string
	Version string
}

// AgentReferenceReader resolves stable facts outside the publication transaction.
// F is the typed immutable fact model settled by T5. No provider/model readiness,
// execution initiation, or volatile availability promise belongs to this port.
type AgentReferenceReader[F any] interface {
	// Resolve returns detached facts for exactly the requested immutable version.
	// Missing is (zero, false, nil); dependency failure is an error, not a validation
	// finding. Stored identities cannot silently resolve to newer reference versions.
	Resolve(context.Context, AgentReference) (F, bool, error)
}
