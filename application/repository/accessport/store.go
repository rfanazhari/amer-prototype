package accessport

import (
	"context"
	"time"

	"github.com/rfanazhari/amer-prototype/domain/enum"
)

// AccountRecord is the detached administrative identity persisted independently of the domain aggregate.
type AccountRecord struct {
	ID           string
	Name         string
	Email        string
	PasswordHash string
	Role         enum.Role
	Status       enum.AccountStatus
	CreatedBy    *string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// SessionRecord is the detached administrative session lifetime persisted independently of the domain aggregate.
type SessionRecord struct {
	ID        string
	AccountID string
	TokenHash string
	CreatedAt time.Time
	ExpiresAt time.Time
	RevokedAt *time.Time
}

// Accounts persists administrative identity with a serialized uniqueness boundary.
type Accounts interface {
	// WithinAccounts serializes bootstrap and provision writes. Callback or known transaction failure rolls back.
	WithinAccounts(ctx context.Context, fn func(AccountTx) error) error
	// FindByID restores one account by stable identity or returns ErrNotFound.
	FindByID(ctx context.Context, id string) (AccountRecord, error)
	// FindByEmail restores one account by canonical email or returns ErrNotFound.
	FindByEmail(ctx context.Context, email string) (AccountRecord, error)
}

// AccountTx exposes account persistence inside a uniqueness transaction.
type AccountTx interface {
	// Count returns the number of administrative accounts currently stored.
	Count(ctx context.Context) (int64, error)
	// FindByEmail restores a matching account without treating absence as an error.
	FindByEmail(ctx context.Context, email string) (AccountRecord, bool, error)
	// Insert stores a new account or returns AccountConflictError for a duplicate canonical email.
	Insert(ctx context.Context, record AccountRecord) error
}

// Sessions persists administrative session lifetime with explicit revocation writes.
type Sessions interface {
	// Insert stores a new session or returns TokenConflictError for a duplicate digest.
	Insert(ctx context.Context, record SessionRecord) error
	// FindByTokenHash restores one session by digest or returns ErrNotFound.
	FindByTokenHash(ctx context.Context, tokenHash string) (SessionRecord, error)
	// FindWithAccountByTokenHash restores the session and its bound account in one read or returns ErrNotFound.
	FindWithAccountByTokenHash(ctx context.Context, tokenHash string) (SessionRecord, AccountRecord, error)
	// UpdateRevocation writes revoked_at explicitly, including a SQL NULL when the pointer is nil.
	UpdateRevocation(ctx context.Context, id string, revokedAt *time.Time) error
}
