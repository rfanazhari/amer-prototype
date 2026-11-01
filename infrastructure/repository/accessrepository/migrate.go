package accessrepository

import (
	"errors"

	"gorm.io/gorm"
)

// constraintStatements install additive uniqueness, check, index and restrict constraints idempotently.
var constraintStatements = []string{
	`CREATE UNIQUE INDEX IF NOT EXISTS "administrative_accounts_email_INDEX" ON administrative_accounts (email)`,
	`CREATE INDEX IF NOT EXISTS "administrative_accounts_created_by_INDEX" ON administrative_accounts (created_by)`,
	`CREATE UNIQUE INDEX IF NOT EXISTS "administrative_sessions_token_hash_INDEX" ON administrative_sessions (token_hash)`,
	`CREATE INDEX IF NOT EXISTS "administrative_sessions_account_id_INDEX" ON administrative_sessions (account_id)`,
	`CREATE INDEX IF NOT EXISTS "administrative_sessions_expires_at_INDEX" ON administrative_sessions (expires_at)`,
	`DO $$ BEGIN ALTER TABLE administrative_accounts ADD CONSTRAINT administrative_accounts_role_check CHECK (role IN ('ADMIN','HR')); EXCEPTION WHEN duplicate_object THEN NULL; END $$`,
	`DO $$ BEGIN ALTER TABLE administrative_accounts ADD CONSTRAINT administrative_accounts_status_check CHECK (status IN ('ACTIVE','DISABLED')); EXCEPTION WHEN duplicate_object THEN NULL; END $$`,
	`DO $$ BEGIN ALTER TABLE administrative_accounts ADD CONSTRAINT administrative_accounts_password_hash_check CHECK (char_length(password_hash) > 0); EXCEPTION WHEN duplicate_object THEN NULL; END $$`,
	`DO $$ BEGIN ALTER TABLE administrative_accounts ADD CONSTRAINT administrative_accounts_created_by_fkey FOREIGN KEY (created_by) REFERENCES administrative_accounts(id) ON DELETE RESTRICT ON UPDATE RESTRICT; EXCEPTION WHEN duplicate_object THEN NULL; END $$`,
	`DO $$ BEGIN ALTER TABLE administrative_sessions ADD CONSTRAINT administrative_sessions_token_hash_check CHECK (token_hash ~ '^[0-9a-f]{64}$'); EXCEPTION WHEN duplicate_object THEN NULL; END $$`,
	`DO $$ BEGIN ALTER TABLE administrative_sessions ADD CONSTRAINT administrative_sessions_expiry_check CHECK (expires_at > created_at); EXCEPTION WHEN duplicate_object THEN NULL; END $$`,
	`DO $$ BEGIN ALTER TABLE administrative_sessions ADD CONSTRAINT administrative_sessions_revoked_at_check CHECK (revoked_at IS NULL OR revoked_at >= created_at); EXCEPTION WHEN duplicate_object THEN NULL; END $$`,
	`DO $$ BEGIN ALTER TABLE administrative_sessions ADD CONSTRAINT administrative_sessions_account_id_fkey FOREIGN KEY (account_id) REFERENCES administrative_accounts(id) ON DELETE RESTRICT ON UPDATE RESTRICT; EXCEPTION WHEN duplicate_object THEN NULL; END $$`,
}

// AutoMigrate installs the two administrative tables and their named constraints without candidate or assignment models.
func AutoMigrate(db *gorm.DB) error {
	if db == nil {
		return errors.New("database is required")
	}
	if err := db.AutoMigrate(&AccountModel{}, &SessionModel{}); err != nil {
		return err
	}
	for _, statement := range constraintStatements {
		if err := db.Exec(statement).Error; err != nil {
			return err
		}
	}
	return nil
}
