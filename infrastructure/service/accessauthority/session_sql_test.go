package accessauthority_test

import (
	"context"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/rfanazhari/amer-prototype/application/service"
	"github.com/rfanazhari/amer-prototype/domain/enum"
	"github.com/rfanazhari/amer-prototype/infrastructure/repository/accessrepository"
	"github.com/rfanazhari/amer-prototype/infrastructure/service/accessauthority"
	"github.com/rfanazhari/amer-prototype/test/mocks"
	"github.com/rfanazhari/amer-prototype/test/testdata"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// sqlSessionCase names one SQL-backed current-session resolution outcome.
type sqlSessionCase struct {
	name     string
	role     string
	status   string
	expires  time.Time
	revoked  *time.Time
	noCookie bool
	noRows   bool
	wantErr  error
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

// expectSessionAccountQuery prepares the joined administrative session lookup used by the resolver.
func expectSessionAccountQuery(t *testing.T, mock sqlmock.Sqlmock, role, status string, expires time.Time, revoked *time.Time, noRows bool) {
	t.Helper()
	columns := []string{
		"id", "account_id", "token_hash", "created_at", "expires_at", "revoked_at",
		"account_pk", "name", "email", "password_hash", "role", "status", "created_by", "account_created_at", "updated_at",
	}
	if noRows {
		mock.ExpectQuery(`(?i)SELECT .* FROM .*administrative_sessions`).WillReturnRows(sqlmock.NewRows(columns))
		return
	}
	account := testdata.AdministrativeAccountRecord(t)
	session := testdata.AdministrativeSessionRecord(t)
	rows := sqlmock.NewRows(columns).AddRow(
		session.ID, session.AccountID, testdata.TokenHash, session.CreatedAt, expires, revoked,
		account.ID, account.Name, account.Email, account.PasswordHash, role, status, nil, account.CreatedAt, account.UpdatedAt,
	)
	mock.ExpectQuery(`(?i)SELECT .* FROM .*administrative_sessions`).WithArgs(testdata.TokenHash).WillReturnRows(rows)
}

// TestSessionsResolveSQL uses sqlmock to prove expiry, revocation, disablement and missing credentials.
func TestSessionsResolveSQL(t *testing.T) {
	revokedAt := testdata.Now().Add(time.Minute)
	tests := []sqlSessionCase{
		{name: "current hr session", role: enum.HR.String(), status: enum.Active.String(), expires: testdata.Now().Add(2 * time.Hour)},
		{name: "expired session", role: enum.HR.String(), status: enum.Active.String(), expires: testdata.Now().Add(time.Hour), wantErr: service.ErrSessionUnbound},
		{name: "revoked session", role: enum.HR.String(), status: enum.Active.String(), expires: testdata.Now().Add(2 * time.Hour), revoked: &revokedAt, wantErr: service.ErrSessionUnbound},
		{name: "disabled account", role: enum.HR.String(), status: enum.Disabled.String(), expires: testdata.Now().Add(2 * time.Hour), wantErr: service.ErrSessionUnbound},
		{name: "unknown digest", role: enum.HR.String(), status: enum.Active.String(), expires: testdata.Now().Add(2 * time.Hour), noRows: true, wantErr: service.ErrSessionUnbound},
		{name: "missing cookie", noCookie: true, wantErr: service.ErrSessionUnbound},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store, sqlMock := sqlStore(t)
			tokens := mocks.NewTokenSecrets(t)
			clock := mocks.NewClock(t)
			if !tt.noCookie {
				tokens.On("Digest", service.CredentialAdministrative, "plain-session-secret").Return(testdata.TokenHash, nil).Once()
				clock.On("Now").Return(testdata.Now().Add(time.Hour)).Maybe()
				expectSessionAccountQuery(t, sqlMock, tt.role, tt.status, tt.expires, tt.revoked, tt.noRows)
			}
			resolver := accessauthority.Sessions{Store: store, Tokens: tokens, Clock: clock}
			ctx := context.Background()
			if !tt.noCookie {
				ctx = service.WithAdministrativeCredential(ctx, "plain-session-secret")
			}
			session, account, err := resolver.Resolve(ctx)
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, testdata.ID, account.ID)
			assert.Equal(t, testdata.SessionID, session.ID)
			assert.Equal(t, enum.HR, account.Role)
		})
	}
}
