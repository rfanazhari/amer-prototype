package accessusecase_test

import (
	"context"
	"errors"
	"testing"

	"github.com/rfanazhari/amer-prototype/application/repository/accessport"
	"github.com/rfanazhari/amer-prototype/application/service"
	"github.com/rfanazhari/amer-prototype/application/usecase/accessusecase"
	"github.com/rfanazhari/amer-prototype/domain/enum"
	"github.com/rfanazhari/amer-prototype/test/mocks"
	"github.com/rfanazhari/amer-prototype/test/testdata"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// bootstrapCase names one bootstrap initialization outcome.
type bootstrapCase struct {
	name       string
	email      string
	password   string
	count      int64
	existing   accessport.AccountRecord
	found      bool
	insertErr  error
	txErr      error
	wantErr    error
	wantInsert bool
	wantSkip   bool
}

// TestBootstrapInitializesFirstAdminWithoutOverwrite covers empty, restart, incompatible and conflict paths.
func TestBootstrapInitializesFirstAdminWithoutOverwrite(t *testing.T) {
	admin := testdata.AdministrativeAccountRecord(t)
	hr := testdata.AdministrativeAccountRecord(t)
	hr.Role = enum.HR
	disabled := testdata.AdministrativeAccountRecord(t)
	disabled.Status = enum.Disabled
	other := testdata.AdministrativeAccountRecord(t)
	other.Email = "rina@example.com"
	storeFailure := errors.New("accounts unavailable")
	tests := []bootstrapCase{
		{name: "empty database inserts admin", email: "Sinta@Example.com", password: "correct-horse", count: 0, wantInsert: true},
		{name: "matching active admin skips overwrite", email: "sinta@example.com", password: "correct-horse", count: 1, existing: admin, found: true, wantSkip: true},
		{name: "matching hr fails startup", email: "sinta@example.com", password: "correct-horse", count: 1, existing: hr, found: true, wantErr: service.ErrBootstrapIncompatible},
		{name: "matching disabled fails startup", email: "sinta@example.com", password: "correct-horse", count: 1, existing: disabled, found: true, wantErr: service.ErrBootstrapIncompatible},
		{name: "initialized without env email skips", email: "missing@example.com", password: "correct-horse", count: 1, existing: other, found: false, wantSkip: true},
		{name: "invalid password rejects before hash", email: "sinta@example.com", password: "short", wantErr: service.ErrInvalidInput},
		{name: "known persistence failure is not success", email: "sinta@example.com", password: "correct-horse", count: 0, txErr: storeFailure, wantErr: storeFailure, wantInsert: true},
		{name: "unique race collapses to existing admin", email: "sinta@example.com", password: "correct-horse", count: 0, existing: admin, found: true, insertErr: accessport.AccountConflictError{}, wantSkip: true, wantInsert: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			accounts := mocks.NewAccounts(t)
			hasher := mocks.NewPasswordHasher(t)
			clock := mocks.NewClock(t)
			op := accessusecase.Bootstrap{
				Accounts: accounts,
				Hasher:   hasher,
				Clock:    clock,
				Name:     " Sinta Wijaya ",
				Email:    tt.email,
				Password: tt.password,
			}
			if tt.wantErr != service.ErrInvalidInput {
				clock.On("Now").Return(testdata.Now()).Maybe()
				hasher.On("Hash", mock.Anything, tt.password).Return(testdata.PasswordHash, nil).Once()
				accounts.On("WithinAccounts", mock.Anything, mock.Anything).Return(func(ctx context.Context, fn func(accessport.AccountTx) error) error {
					tx := mocks.NewAccountTx(t)
					tx.On("Count", mock.Anything).Return(tt.count, nil).Once()
					if tt.count > 0 {
						tx.On("FindByEmail", mock.Anything, mock.Anything).Return(tt.existing, tt.found, nil).Once()
					}
					if tt.wantInsert {
						tx.On("Insert", mock.Anything, mock.MatchedBy(func(record accessport.AccountRecord) bool {
							return record.Email == "sinta@example.com" &&
								record.Name == "Sinta Wijaya" &&
								record.Role == enum.Admin &&
								record.Status == enum.Active &&
								record.CreatedBy == nil &&
								record.PasswordHash == testdata.PasswordHash &&
								record.PasswordHash != tt.password
						})).Return(tt.insertErr).Once()
						if tt.insertErr != nil {
							tx.On("FindByEmail", mock.Anything, "sinta@example.com").Return(tt.existing, tt.found, nil).Once()
						}
					}
					if tt.txErr != nil {
						_ = fn(tx)
						return tt.txErr
					}
					return fn(tx)
				}).Once()
			}
			err := op.Execute(context.Background())
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				assert.NotContains(t, err.Error(), tt.password)
				return
			}
			require.NoError(t, err)
		})
	}
}
