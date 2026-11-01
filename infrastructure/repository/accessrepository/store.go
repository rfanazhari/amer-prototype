package accessrepository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/rfanazhari/amer-prototype/application/repository/accessport"
	"gorm.io/gorm"
)

// Store implements administrative account and session persistence through a runtime-owned GORM pool.
type Store struct{ DB *gorm.DB }

// accountTx holds the serialized uniqueness transaction for bootstrap and HR provision.
type accountTx struct{ tx *gorm.DB }

// loginTx holds the locked-account login transaction used to recheck eligibility before insert.
type loginTx struct {
	tx        *gorm.DB
	accountID string
}

// classifyUnique maps a PostgreSQL unique-violation onto a classified access conflict.
func classifyUnique(err error, conflict error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return conflict
	}
	return err
}

// notFoundOrError translates GORM absence into ErrNotFound while preserving other storage failures.
func notFoundOrError(err error) error {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return accessport.ErrNotFound
	}
	return err
}

// WithinLogin locks the account row, runs the callback, and classifies a post-callback commit failure as uncertain.
func (s Store) WithinLogin(ctx context.Context, accountID string, fn func(accessport.LoginTx) error) error {
	var committedCallback bool
	err := s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec("SELECT id FROM administrative_accounts WHERE id = ? FOR UPDATE", accountID).Error; err != nil {
			return err
		}
		if err := fn(loginTx{tx: tx, accountID: accountID}); err != nil {
			return err
		}
		committedCallback = true
		return nil
	})
	if err != nil && committedCallback {
		return &accessport.CommitUncertainError{Cause: err}
	}
	return err
}

// Account restores the locked account for eligibility and stored-hash recheck.
func (t loginTx) Account(ctx context.Context) (accessport.AccountRecord, error) {
	var model AccountModel
	if err := t.tx.WithContext(ctx).Where("id = ?", t.accountID).Take(&model).Error; err != nil {
		return accessport.AccountRecord{}, notFoundOrError(err)
	}
	return toAccount(model)
}

// InsertSession stores the session in the open transaction or classifies a duplicate digest.
func (t loginTx) InsertSession(ctx context.Context, record accessport.SessionRecord) error {
	model := fromSession(record)
	if err := t.tx.WithContext(ctx).Create(&model).Error; err != nil {
		return classifyUnique(err, accessport.TokenConflictError{})
	}
	return nil
}

// WithinAccounts serializes account writes with an exclusive table lock and rolls back on callback failure.
func (s Store) WithinAccounts(ctx context.Context, fn func(accessport.AccountTx) error) error {
	return s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec("LOCK TABLE administrative_accounts IN EXCLUSIVE MODE").Error; err != nil {
			return err
		}
		return fn(accountTx{tx: tx})
	})
}

// FindByID restores one account by identity or returns ErrNotFound.
func (s Store) FindByID(ctx context.Context, id string) (accessport.AccountRecord, error) {
	var model AccountModel
	if err := s.DB.WithContext(ctx).Where("id = ?", id).Take(&model).Error; err != nil {
		return accessport.AccountRecord{}, notFoundOrError(err)
	}
	return toAccount(model)
}

// FindByEmail restores one account by canonical email or returns ErrNotFound.
func (s Store) FindByEmail(ctx context.Context, email string) (accessport.AccountRecord, error) {
	var model AccountModel
	if err := s.DB.WithContext(ctx).Where("email = ?", email).Take(&model).Error; err != nil {
		return accessport.AccountRecord{}, notFoundOrError(err)
	}
	return toAccount(model)
}

// Insert stores a new account or classifies a duplicate canonical email as AccountConflictError.
func (t accountTx) Insert(ctx context.Context, record accessport.AccountRecord) error {
	model := fromAccount(record)
	if err := t.tx.WithContext(ctx).Create(&model).Error; err != nil {
		return classifyUnique(err, accessport.AccountConflictError{})
	}
	return nil
}

// Count returns the number of administrative accounts currently stored.
func (t accountTx) Count(ctx context.Context) (int64, error) {
	var n int64
	err := t.tx.WithContext(ctx).Model(&AccountModel{}).Count(&n).Error
	return n, err
}

// FindByEmail restores a matching account without treating absence as an error.
func (t accountTx) FindByEmail(ctx context.Context, email string) (accessport.AccountRecord, bool, error) {
	var model AccountModel
	err := t.tx.WithContext(ctx).Where("email = ?", email).Take(&model).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return accessport.AccountRecord{}, false, nil
	}
	if err != nil {
		return accessport.AccountRecord{}, false, err
	}
	record, err := toAccount(model)
	return record, err == nil, err
}

// Insert stores a new session inside a transaction or classifies a duplicate digest as TokenConflictError.
func (s Store) Insert(ctx context.Context, record accessport.SessionRecord) error {
	return s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		model := fromSession(record)
		if err := tx.Create(&model).Error; err != nil {
			return classifyUnique(err, accessport.TokenConflictError{})
		}
		return nil
	})
}

// FindByTokenHash restores one session by digest or returns ErrNotFound.
func (s Store) FindByTokenHash(ctx context.Context, tokenHash string) (accessport.SessionRecord, error) {
	var model SessionModel
	if err := s.DB.WithContext(ctx).Where("token_hash = ?", tokenHash).Take(&model).Error; err != nil {
		return accessport.SessionRecord{}, notFoundOrError(err)
	}
	return toSession(model)
}

// FindWithAccountByTokenHash restores the session and bound account in one joined read or returns ErrNotFound.
func (s Store) FindWithAccountByTokenHash(ctx context.Context, tokenHash string) (accessport.SessionRecord, accessport.AccountRecord, error) {
	var session SessionModel
	var account AccountModel
	err := s.DB.WithContext(ctx).Table("administrative_sessions AS s").
		Select("s.id, s.account_id, s.token_hash, s.created_at, s.expires_at, s.revoked_at, a.id AS account_pk, a.name, a.email, a.password_hash, a.role, a.status, a.created_by, a.created_at AS account_created_at, a.updated_at").
		Joins("JOIN administrative_accounts AS a ON a.id = s.account_id").
		Where("s.token_hash = ?", tokenHash).
		Row().
		Scan(
			&session.ID, &session.AccountID, &session.TokenHash, &session.CreatedAt, &session.ExpiresAt, &session.RevokedAt,
			&account.ID, &account.Name, &account.Email, &account.PasswordHash, &account.Role, &account.Status, &account.CreatedBy, &account.CreatedAt, &account.UpdatedAt,
		)
	if errors.Is(err, gorm.ErrRecordNotFound) || errors.Is(err, sql.ErrNoRows) {
		return accessport.SessionRecord{}, accessport.AccountRecord{}, accessport.ErrNotFound
	}
	if err != nil {
		return accessport.SessionRecord{}, accessport.AccountRecord{}, err
	}
	sessionRecord, err := toSession(session)
	if err != nil {
		return accessport.SessionRecord{}, accessport.AccountRecord{}, err
	}
	accountRecord, err := toAccount(account)
	return sessionRecord, accountRecord, err
}

// UpdateRevocation writes revoked_at explicitly, including a SQL NULL when the pointer is nil.
func (s Store) UpdateRevocation(ctx context.Context, id string, revokedAt *time.Time) error {
	return s.DB.WithContext(ctx).Model(&SessionModel{}).Where("id = ?", id).Select("revoked_at").Updates(map[string]any{"revoked_at": copyTime(revokedAt)}).Error
}
