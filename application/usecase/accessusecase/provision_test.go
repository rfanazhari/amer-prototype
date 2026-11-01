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

// provisionCase names one ADMIN-only HR creation outcome.
type provisionCase struct {
	name      string
	actor     accessport.AccountRecord
	insertErr error
	txErr     error
	wantErr   error
	wantWrite bool
}

// TestProvisionHRRequiresCurrentAdminAndUniqueEmail denies HR actors and classifies conflicts.
func TestProvisionHRRequiresCurrentAdminAndUniqueEmail(t *testing.T) {
	admin := testdata.AdministrativeAccountRecord(t)
	hr := testdata.AdministrativeAccountRecord(t)
	hr.Role = enum.HR
	session := testdata.AdministrativeSessionRecord(t)
	storeFailure := errors.New("accounts unavailable")
	tests := []provisionCase{
		{name: "admin provisions hr", actor: admin, wantWrite: true},
		{name: "hr cannot provision", actor: hr, wantErr: service.ErrForbidden},
		{name: "duplicate email is conflict", actor: admin, insertErr: accessport.AccountConflictError{}, wantErr: accessport.AccountConflictError{}, wantWrite: true},
		{name: "known persistence failure is not success", actor: admin, txErr: storeFailure, wantErr: storeFailure, wantWrite: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			accounts := mocks.NewAccounts(t)
			current := mocks.NewCurrentAdministrativeSession(t)
			hasher := mocks.NewPasswordHasher(t)
			clock := mocks.NewClock(t)
			actorSession := session
			actorSession.AccountID = tt.actor.ID
			current.On("Resolve", mock.Anything).Return(actorSession, tt.actor, nil).Once()
			if tt.wantWrite {
				clock.On("Now").Return(testdata.Now())
				hasher.On("Hash", mock.Anything, "correct-horse").Return(testdata.PasswordHash, nil).Once()
				accounts.On("WithinAccounts", mock.Anything, mock.Anything).Return(func(ctx context.Context, fn func(accessport.AccountTx) error) error {
					tx := mocks.NewAccountTx(t)
					tx.On("FindByEmail", mock.Anything, admin.Email).Return(admin, true, nil).Once()
					tx.On("Insert", mock.Anything, mock.MatchedBy(func(record accessport.AccountRecord) bool {
						return record.Name == "Sinta Wijaya" &&
							record.Email == "sinta@example.com" &&
							record.Role == enum.HR &&
							record.Status == enum.Active &&
							record.PasswordHash == testdata.PasswordHash &&
							record.CreatedBy != nil &&
							*record.CreatedBy == admin.ID
					})).Return(tt.insertErr).Once()
					if tt.txErr != nil {
						_ = fn(tx)
						return tt.txErr
					}
					return fn(tx)
				}).Once()
			}
			got, err := (accessusecase.ProvisionHR{
				Accounts: accounts,
				Current:  current,
				Hasher:   hasher,
				Clock:    clock,
			}).Execute(context.Background(), " Sinta Wijaya ", "Sinta@Example.com", "correct-horse")
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				assert.Nil(t, got)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, got)
			assert.Equal(t, "Sinta Wijaya", got.Name)
			assert.Equal(t, "sinta@example.com", got.Email)
			assert.Equal(t, enum.HR.String(), got.Role)
			assert.Equal(t, enum.Active.String(), got.Status)
			assert.NotEmpty(t, got.ID)
			assert.NotEqual(t, "correct-horse", got.ID)
		})
	}
}
