package agentconfigport

import "context"

// CommandIdentity contains the already-authorized, normalized identity selected
// by the eventual Q1 contract. These fields do not define public key syntax or scope.
type CommandIdentity struct {
	Scope string
	Kind  string
	Key   string
}

// CommandResult retains only intent identity and the original revision reference;
// configuration bodies and confidential reference contents must never be included.
type CommandResult struct {
	Fingerprint string
	InstanceID  string
	RevisionID  string
}

// Reader provides bounded detached reads, without loading revision history.
// I and R will be the domain snapshots settled by T3; no configuration schema
// or transport DTO is selected here. Callers must authorize before every read.
type Reader[I, R any] interface {
	// FindInstance returns a detached instance; missing records return a classified error.
	FindInstance(context.Context, string) (I, error)
	// FindRevision restores one owned immutable revision without current-policy revalidation.
	FindRevision(context.Context, string, string) (R, error)
}

// ReplayReader reconciles a command using a fresh transaction/read after a unique
// conflict or CommitUncertainError. Absence is distinct from an unavailable store.
// A matching result is the original outcome, even after later publications.
type ReplayReader interface {
	// FindCommand returns a detached result and explicit presence, after authorization.
	// The caller compares fingerprint and instance ownership before disclosing a revision.
	FindCommand(context.Context, CommandIdentity) (CommandResult, bool, error)
}

// Mutation provides atomic creation and serialized instance mutations. Type
// parameters defer concrete domain types to T3 and Q1 rather than guessing DTOs.
type Mutation[I, V, R any] interface {
	// Create atomically inserts the instance and owned initial draft, with no publication.
	Create(context.Context, I) error
	// WithinInstance locks the instance before draft/validation/revision rows and
	// invokes the callback once. Callback error or known transaction failure rolls
	// back all writes. Nil means durable commit; uncertain commit returns
	// CommitUncertainError, never an assertion of rollback. No callback result may
	// escape as success before commit. Unique command losers roll back and reconcile
	// via ReplayReader outside this transaction. The callback must not perform I/O
	// to external references or retain the transaction beyond its lifetime.
	WithinInstance(context.Context, string, func(Transaction[I, V, R]) error) error
}

// Transaction exposes only instance-owned persistence operations. Implementations
// reject cross-instance references. Publication checks FindCommand before current
// draft eligibility, then commits revision, instance pointer, and command together.
// Domain operations own version comparisons, eligibility and active selection.
type Transaction[I, V, R any] interface {
	// Instance returns the locked instance and current draft as a detached domain value.
	Instance(context.Context) (I, error)
	// SaveInstance persists explicit zero/null/empty fields and the aggregate-selected pointer.
	SaveInstance(context.Context, I) error
	// FindValidation loads one immutable assessment belonging to the locked instance.
	FindValidation(context.Context, string) (V, error)
	// InsertValidation records an exact-version assessment after the domain version check.
	InsertValidation(context.Context, V) error
	// FindCommand resolves replay before draft/version checks without changing active selection.
	FindCommand(context.Context, CommandIdentity) (CommandResult, bool, error)
	// InsertRevision inserts an immutable owned revision; revision updates are not exposed.
	InsertRevision(context.Context, R) error
	// InsertCommand stores the unique identity and original result in this transaction.
	InsertCommand(context.Context, CommandIdentity, CommandResult) error
}
