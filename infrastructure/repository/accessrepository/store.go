package accessrepository

import (
	"context"
	"errors"
	"time"

	"github.com/rfanazhari/amer-prototype/application/repository/accessport"
	"gorm.io/gorm"
)

// Store implements administrative account and session persistence through a runtime-owned GORM pool.
type Store struct{ DB *gorm.DB }

// accountTx is an inert uniqueness-transaction handle reserved for GREEN persistence.
type accountTx struct{}

// AutoMigrate installs the two administrative tables. RED leaves schema installation unimplemented.
func AutoMigrate(db *gorm.DB) error {
	if db == nil {
		return errors.New("database is required")
	}
	return errors.New("administrative schema is not installed")
}

// WithinAccounts invokes the callback without a database transaction or uniqueness enforcement.
func (s Store) WithinAccounts(ctx context.Context, fn func(accessport.AccountTx) error) error {
	return fn(accountTx{})
}

// FindByID returns not found until GREEN restoration is implemented.
func (s Store) FindByID(context.Context, string) (accessport.AccountRecord, error) {
	return accessport.AccountRecord{}, accessport.ErrNotFound
}

// FindByEmail returns not found until GREEN restoration is implemented.
func (s Store) FindByEmail(context.Context, string) (accessport.AccountRecord, error) {
	return accessport.AccountRecord{}, accessport.ErrNotFound
}

// Insert accepts the record without writing uniqueness-protected rows.
func (accountTx) Insert(context.Context, accessport.AccountRecord) error { return nil }

// Count reports an empty store until GREEN persistence is implemented.
func (accountTx) Count(context.Context) (int64, error) { return 0, nil }

// FindByEmail reports absence until GREEN restoration is implemented.
func (accountTx) FindByEmail(context.Context, string) (accessport.AccountRecord, bool, error) {
	return accessport.AccountRecord{}, false, nil
}

// Insert accepts the session without writing a unique digest.
func (s Store) Insert(context.Context, accessport.SessionRecord) error { return nil }

// FindByTokenHash returns not found until GREEN restoration is implemented.
func (s Store) FindByTokenHash(context.Context, string) (accessport.SessionRecord, error) {
	return accessport.SessionRecord{}, accessport.ErrNotFound
}

// FindWithAccountByTokenHash returns not found until GREEN joined restoration is implemented.
func (s Store) FindWithAccountByTokenHash(context.Context, string) (accessport.SessionRecord, accessport.AccountRecord, error) {
	return accessport.SessionRecord{}, accessport.AccountRecord{}, accessport.ErrNotFound
}

// UpdateRevocation accepts an explicit revocation write without issuing SQL.
func (s Store) UpdateRevocation(context.Context, string, *time.Time) error { return nil }
