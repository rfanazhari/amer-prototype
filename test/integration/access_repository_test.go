package integration_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/rfanazhari/amer-prototype/application/repository/accessport"
	"github.com/rfanazhari/amer-prototype/domain/enum"
	"github.com/rfanazhari/amer-prototype/infrastructure/repository/accessrepository"
	"github.com/rfanazhari/amer-prototype/infrastructure/repository/taskrepository"
	"github.com/rfanazhari/amer-prototype/test/testdata"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// accessDB opens an authorized integration database without installing access schema.
func accessDB(t *testing.T) *gorm.DB {
	t.Helper()
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL is required")
	}
	db, err := gorm.Open(postgres.Open(url), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	require.NoError(t, err)
	pool, err := db.DB()
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, pool.Close()) })
	return db
}

// accessSchemaCase names one real PostgreSQL proof required by administrative installation.
type accessSchemaCase struct {
	name    string
	verify  func(t *testing.T, db *gorm.DB)
	wantErr bool
}

// TestAdministrativeSchemaAndTransactionsProveTwoTableInstallation covers uniqueness, rollback, restoration and Task coexistence.
func TestAdministrativeSchemaAndTransactionsProveTwoTableInstallation(t *testing.T) {
	tests := []accessSchemaCase{
		{
			name: "exactly two administrative tables with constraints",
			verify: func(t *testing.T, db *gorm.DB) {
				require.NoError(t, db.AutoMigrate(&taskrepository.Model{}))
				require.NoError(t, accessrepository.AutoMigrate(db))
				var tables []string
				require.NoError(t, db.Raw(`SELECT tablename FROM pg_tables WHERE schemaname = 'public' AND tablename LIKE 'administrative%'`).Scan(&tables).Error)
				require.ElementsMatch(t, []string{"administrative_accounts", "administrative_sessions"}, tables)
				var candidates int64
				require.NoError(t, db.Raw(`SELECT COUNT(*) FROM pg_tables WHERE schemaname = 'public' AND tablename IN ('candidates','interview_invitations','candidate_sessions','interview_assignments')`).Scan(&candidates).Error)
				require.Zero(t, candidates)
				assertNamedConstraint(t, db, "administrative_accounts", "administrative_accounts_role_check")
				assertNamedConstraint(t, db, "administrative_accounts", "administrative_accounts_status_check")
				assertNamedIndex(t, db, "administrative_accounts_email_INDEX")
				assertNamedIndex(t, db, "administrative_sessions_token_hash_INDEX")
				assertNamedIndex(t, db, "administrative_sessions_account_id_INDEX")
				assertNamedIndex(t, db, "administrative_sessions_expires_at_INDEX")
			},
		},
		{
			name: "bootstrap and provision uniqueness plus rollback",
			verify: func(t *testing.T, db *gorm.DB) {
				require.NoError(t, accessrepository.AutoMigrate(db))
				store := accessrepository.Store{DB: db}
				ctx := context.Background()
				first := testdata.AdministrativeAccountRecord(t)
				first.ID = uuid.NewString()
				first.Email = "bootstrap-" + uuid.NewString() + "@example.com"
				require.NoError(t, store.WithinAccounts(ctx, func(tx accessport.AccountTx) error {
					return tx.Insert(ctx, first)
				}))
				duplicate := first
				duplicate.ID = uuid.NewString()
				err := store.WithinAccounts(ctx, func(tx accessport.AccountTx) error {
					return tx.Insert(ctx, duplicate)
				})
				var conflict accessport.AccountConflictError
				require.ErrorAs(t, err, &conflict)
				hr := first
				hr.ID = uuid.NewString()
				hr.Email = "hr-" + uuid.NewString() + "@example.com"
				hr.Role = enum.HR
				creator := first.ID
				hr.CreatedBy = &creator
				err = store.WithinAccounts(ctx, func(tx accessport.AccountTx) error {
					require.NoError(t, tx.Insert(ctx, hr))
					return assert.AnError
				})
				require.ErrorIs(t, err, assert.AnError)
				_, err = store.FindByEmail(ctx, hr.Email)
				require.ErrorIs(t, err, accessport.ErrNotFound)
				loaded, err := store.FindByID(ctx, first.ID)
				require.NoError(t, err)
				require.Equal(t, first.Email, loaded.Email)
				require.Nil(t, loaded.CreatedBy)
				require.Equal(t, enum.Admin, loaded.Role)
			},
		},
		{
			name: "stored restoration copies revocation and survives remigrate with tasks",
			verify: func(t *testing.T, db *gorm.DB) {
				require.NoError(t, db.AutoMigrate(&taskrepository.Model{}))
				task := storedTask(t, taskrepository.Store{DB: db}, false)
				require.NoError(t, accessrepository.AutoMigrate(db))
				require.NoError(t, accessrepository.AutoMigrate(db))
				store := accessrepository.Store{DB: db}
				ctx := context.Background()
				account := testdata.AdministrativeAccountRecord(t)
				account.ID = uuid.NewString()
				account.Email = "restore-" + uuid.NewString() + "@example.com"
				require.NoError(t, store.WithinAccounts(ctx, func(tx accessport.AccountTx) error {
					return tx.Insert(ctx, account)
				}))
				session := testdata.AdministrativeSessionRecord(t)
				session.ID = uuid.NewString()
				session.AccountID = account.ID
				session.TokenHash = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
				require.NoError(t, store.Insert(ctx, session))
				revoked := testdata.Now().Add(time.Minute)
				require.NoError(t, store.UpdateRevocation(ctx, session.ID, &revoked))
				loadedSession, loadedAccount, err := store.FindWithAccountByTokenHash(ctx, session.TokenHash)
				require.NoError(t, err)
				require.Equal(t, account.Email, loadedAccount.Email)
				require.Equal(t, session.ID, loadedSession.ID)
				require.NotNil(t, loadedSession.RevokedAt)
				require.True(t, loadedSession.RevokedAt.Equal(revoked.UTC()))
				*loadedSession.RevokedAt = testdata.Now()
				again, err := store.FindByTokenHash(ctx, session.TokenHash)
				require.NoError(t, err)
				require.True(t, again.RevokedAt.Equal(revoked.UTC()))
				foundTask, err := taskrepository.Store{DB: db}.Find(ctx, task.Snapshot().ID)
				require.NoError(t, err)
				require.Equal(t, task.Snapshot().Title, foundTask.Snapshot().Title)
				var pool int
				require.NoError(t, db.Raw(`SHOW max_connections`).Scan(&pool).Error)
				sqlDB, err := db.DB()
				require.NoError(t, err)
				require.LessOrEqual(t, sqlDB.Stats().MaxOpenConnections, 10)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.verify(t, accessDB(t))
		})
	}
}

// assertNamedConstraint requires a table check or foreign-key constraint to be present.
func assertNamedConstraint(t *testing.T, db *gorm.DB, table, name string) {
	t.Helper()
	var found int64
	require.NoError(t, db.Raw(`SELECT COUNT(*) FROM pg_constraint WHERE conname = ? AND conrelid = ?::regclass`, name, table).Scan(&found).Error)
	require.Equal(t, int64(1), found, name)
}

// assertNamedIndex requires a quoted PostgreSQL index name to exist.
func assertNamedIndex(t *testing.T, db *gorm.DB, name string) {
	t.Helper()
	var found int64
	require.NoError(t, db.Raw(`SELECT COUNT(*) FROM pg_class WHERE relkind = 'i' AND relname = ?`, name).Scan(&found).Error)
	require.Equal(t, int64(1), found, name)
}
