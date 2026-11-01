package accessrepository_test

import (
	"context"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/rfanazhari/amer-prototype/application/repository/accessport"
	"github.com/rfanazhari/amer-prototype/infrastructure/repository/accessrepository"
	"github.com/rfanazhari/amer-prototype/pkg/errs"
	"github.com/rfanazhari/amer-prototype/test/testdata"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// persistCase names one administrative SQL behavior that the adapter must perform.
type persistCase struct {
	name string
	run  func(t *testing.T, store accessrepository.Store, mock sqlmock.Sqlmock)
}

// sqlStore opens a GORM adapter over sqlmock with ping disabled so tests own SQL expectations.
func sqlStore(t *testing.T) (accessrepository.Store, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, mock.ExpectationsWereMet())
		_ = db.Close()
	})
	gdb, err := gorm.Open(postgres.New(postgres.Config{Conn: db, DriverName: "postgres"}), &gorm.Config{
		Logger:                 logger.Default.LogMode(logger.Silent),
		SkipDefaultTransaction: true,
		DisableAutomaticPing:   true,
	})
	require.NoError(t, err)
	return accessrepository.Store{DB: gdb}, mock
}

// TestAdministrativePersistenceSQL covers uniqueness, rollback and explicit null writes against mocked PostgreSQL.
func TestAdministrativePersistenceSQL(t *testing.T) {
	account := testdata.AdministrativeAccountRecord(t)
	session := testdata.AdministrativeSessionRecord(t)
	revoked := testdata.Now().Add(time.Minute)
	tests := []persistCase{
		{
			name: "bootstrap insert writes canonical email and null creator",
			run: func(t *testing.T, store accessrepository.Store, mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectExec(`(?i)LOCK TABLE administrative_accounts`).WillReturnResult(sqlmock.NewResult(0, 0))
				mock.ExpectExec(`(?i)INSERT INTO ["']?administrative_accounts["']?`).WithArgs(
					account.ID, account.Name, account.Email, account.PasswordHash, account.Role.String(), account.Status.String(), nil, account.CreatedAt, account.UpdatedAt,
				).WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()
				err := store.WithinAccounts(context.Background(), func(tx accessport.AccountTx) error {
					return tx.Insert(context.Background(), account)
				})
				require.NoError(t, err)
			},
		},
		{
			name: "duplicate email is account conflict",
			run: func(t *testing.T, store accessrepository.Store, mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectExec(`(?i)LOCK TABLE administrative_accounts`).WillReturnResult(sqlmock.NewResult(0, 0))
				mock.ExpectExec(`(?i)INSERT INTO ["']?administrative_accounts["']?`).WillReturnError(&pgconn.PgError{Code: "23505", ConstraintName: "administrative_accounts_email_INDEX"})
				mock.ExpectRollback()
				err := store.WithinAccounts(context.Background(), func(tx accessport.AccountTx) error {
					return tx.Insert(context.Background(), account)
				})
				var conflict accessport.AccountConflictError
				require.ErrorAs(t, err, &conflict)
			},
		},
		{
			name: "callback failure rolls back account insert",
			run: func(t *testing.T, store accessrepository.Store, mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectExec(`(?i)LOCK TABLE administrative_accounts`).WillReturnResult(sqlmock.NewResult(0, 0))
				mock.ExpectExec(`(?i)INSERT INTO ["']?administrative_accounts["']?`).WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectRollback()
				err := store.WithinAccounts(context.Background(), func(tx accessport.AccountTx) error {
					require.NoError(t, tx.Insert(context.Background(), account))
					return assert.AnError
				})
				require.ErrorIs(t, err, assert.AnError)
			},
		},
		{
			name: "session revocation writes explicit timestamp",
			run: func(t *testing.T, store accessrepository.Store, mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectExec(`(?i)INSERT INTO ["']?administrative_sessions["']?`).WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()
				mock.ExpectExec(`(?i)UPDATE ["']?administrative_sessions["']? SET .*revoked_at`).WithArgs(revoked, session.ID).WillReturnResult(sqlmock.NewResult(0, 1))
				require.NoError(t, store.Insert(context.Background(), session))
				require.NoError(t, store.UpdateRevocation(context.Background(), session.ID, &revoked))
			},
		},
		{
			name: "null revocation update includes SQL NULL",
			run: func(t *testing.T, store accessrepository.Store, mock sqlmock.Sqlmock) {
				mock.ExpectExec(`(?i)UPDATE ["']?administrative_sessions["']? SET .*revoked_at`).WithArgs(nil, session.ID).WillReturnResult(sqlmock.NewResult(0, 1))
				require.NoError(t, store.UpdateRevocation(context.Background(), session.ID, nil))
			},
		},
		{
			name: "corrupt stored role is technical failure",
			run: func(t *testing.T, store accessrepository.Store, mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id", "name", "email", "password_hash", "role", "status", "created_by", "created_at", "updated_at"}).
					AddRow(account.ID, account.Name, account.Email, account.PasswordHash, "SUPERUSER", account.Status.String(), nil, account.CreatedAt, account.UpdatedAt)
				mock.ExpectQuery(`(?i)SELECT .* FROM ["']?administrative_accounts["']?`).WillReturnRows(rows)
				_, err := store.FindByID(context.Background(), account.ID)
				var technical errs.Technical
				require.ErrorAs(t, err, &technical)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store, mock := sqlStore(t)
			tt.run(t, store, mock)
		})
	}
}
