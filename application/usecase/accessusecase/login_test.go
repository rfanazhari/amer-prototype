package accessusecase_test

import (
	"context"
	"testing"
	"time"

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

// loginCase names one administrative login proof and commit outcome.
type loginCase struct {
	name      string
	email     string
	password  string
	found     bool
	account   accessport.AccountRecord
	locked    accessport.AccountRecord
	verifyErr error
	commitErr error
	wantErr   error
	wantDummy bool
	wantLogin bool
}

// TestLoginRechecksEligibilityBeforeSessionCommit withholds metadata on proof and commit failure.
func TestLoginRechecksEligibilityBeforeSessionCommit(t *testing.T) {
	account := testdata.AdministrativeAccountRecord(t)
	disabled := testdata.AdministrativeAccountRecord(t)
	disabled.Status = enum.Disabled
	stale := testdata.AdministrativeAccountRecord(t)
	stale.PasswordHash = testdata.PasswordHash + "-stale"
	uncertain := &accessport.CommitUncertainError{Cause: context.DeadlineExceeded}
	tests := []loginCase{
		{name: "committed login", email: "Sinta@Example.com", password: "correct-horse", found: true, account: account, locked: account, wantLogin: true},
		{name: "unknown account uses dummy proof", email: "missing@example.com", password: "correct-horse", wantDummy: true, wantErr: service.ErrAuthenticationFailed},
		{name: "wrong password", email: "sinta@example.com", password: "correct-horse", found: true, account: account, verifyErr: service.ErrAuthenticationFailed, wantErr: service.ErrAuthenticationFailed},
		{name: "disabled account", email: "sinta@example.com", password: "correct-horse", found: true, account: disabled, verifyErr: nil, wantErr: service.ErrAuthenticationFailed},
		{name: "stale hash after lock", email: "sinta@example.com", password: "correct-horse", found: true, account: account, locked: stale, wantErr: service.ErrAuthenticationFailed},
		{name: "empty password is invalid input", email: "sinta@example.com", password: "", wantErr: service.ErrInvalidInput},
		{name: "uncertain commit is not success", email: "sinta@example.com", password: "correct-horse", found: true, account: account, locked: account, commitErr: uncertain, wantErr: uncertain},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			accounts := mocks.NewAccounts(t)
			issuance := mocks.NewSessionIssuance(t)
			hasher := mocks.NewPasswordHasher(t)
			tokens := mocks.NewTokenSecrets(t)
			clock := mocks.NewClock(t)
			if tt.wantErr != service.ErrInvalidInput {
				canonical := "sinta@example.com"
				if tt.wantDummy {
					canonical = "missing@example.com"
				}
				accounts.On("FindByEmail", mock.Anything, canonical).Return(tt.account, func() error {
					if tt.found {
						return nil
					}
					return accessport.ErrNotFound
				}()).Once()
				if tt.wantDummy {
					hasher.On("Verify", mock.Anything, service.DummyEncodedPassword, tt.password).Return(service.ErrAuthenticationFailed).Once()
				} else {
					hasher.On("Verify", mock.Anything, tt.account.PasswordHash, tt.password).Return(tt.verifyErr).Once()
				}
			}
			if tt.wantLogin || tt.commitErr != nil || (tt.found && tt.verifyErr == nil && tt.account.Status == enum.Active && tt.locked.PasswordHash != account.PasswordHash) {
				if tt.wantLogin || tt.commitErr != nil || tt.locked.PasswordHash != account.PasswordHash {
					clock.On("Now").Return(testdata.Now())
					tokens.On("Issue", mock.Anything, service.CredentialAdministrative).Return("plain-session-secret", testdata.TokenHash, nil).Once()
					issuance.On("WithinLogin", mock.Anything, account.ID, mock.Anything).Return(func(ctx context.Context, accountID string, fn func(accessport.LoginTx) error) error {
						tx := mocks.NewLoginTx(t)
						tx.On("Account", mock.Anything).Return(tt.locked, nil).Once()
						if tt.locked.PasswordHash == account.PasswordHash && tt.locked.Status == enum.Active {
							tx.On("InsertSession", mock.Anything, mock.MatchedBy(func(record accessport.SessionRecord) bool {
								return record.AccountID == account.ID &&
									record.TokenHash == testdata.TokenHash &&
									record.ExpiresAt.Equal(testdata.Now().Add(2*time.Hour)) &&
									record.RevokedAt == nil
							})).Return(nil).Once()
						}
						if tt.commitErr != nil {
							_ = fn(tx)
							return tt.commitErr
						}
						return fn(tx)
					}).Once()
				}
			}
			got, err := (accessusecase.Login{
				Accounts: accounts,
				Issuance: issuance,
				Hasher:   hasher,
				Tokens:   tokens,
				Clock:    clock,
			}).Execute(context.Background(), tt.email, tt.password)
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				assert.Nil(t, got)
				if tt.password != "" {
					assert.NotContains(t, err.Error(), tt.password)
				}
				assert.NotContains(t, err.Error(), "plain-session-secret")
				return
			}
			require.NoError(t, err)
			require.NotNil(t, got)
			assert.Equal(t, account.ID, got.Account.ID)
			assert.Equal(t, "sinta@example.com", got.Account.Email)
			assert.Equal(t, testdata.Now().Add(2*time.Hour), got.Session.ExpiresAt)
			assert.NotEmpty(t, got.Session.ID)
			assert.NotEqual(t, "plain-session-secret", got.Session.ID)
			assert.Equal(t, "plain-session-secret", got.SessionToken)
			assert.NotContains(t, got.Account.Name, testdata.PasswordHash)
		})
	}
}
