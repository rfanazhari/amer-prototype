package integration_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/rfanazhari/amer-prototype/application/repository/accessport"
	"github.com/rfanazhari/amer-prototype/infrastructure/repository/accessrepository"
	"github.com/rfanazhari/amer-prototype/test/testdata"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// replacementAccount stores one active administrative account for session replacement proofs.
func replacementAccount(t *testing.T, store accessrepository.Store) accessport.AccountRecord {
	t.Helper()
	ctx := context.Background()
	account := testdata.AdministrativeAccountRecord(t)
	account.ID = uuid.NewString()
	account.Email = "replacement-" + uuid.NewString() + "@example.com"
	require.NoError(t, store.WithinAccounts(ctx, func(tx accessport.AccountTx) error { return tx.Insert(ctx, account) }))
	return account
}

// newReplacementSession builds a fresh live session record with a unique digest for the account.
func newReplacementSession(accountID string, now time.Time) accessport.SessionRecord {
	digest := sha256.Sum256([]byte(uuid.NewString()))
	return accessport.SessionRecord{
		ID:        uuid.NewString(),
		AccountID: accountID,
		TokenHash: hex.EncodeToString(digest[:]),
		CreatedAt: now,
		ExpiresAt: now.Add(2 * time.Hour),
	}
}

// loginAs runs the login transaction shape: replace the account's other sessions, then insert the new one.
func loginAs(ctx context.Context, store accessrepository.Store, record accessport.SessionRecord) error {
	return store.WithinLogin(ctx, record.AccountID, func(tx accessport.LoginTx) error {
		if err := tx.DeleteOtherSessions(ctx, record.ID); err != nil {
			return err
		}
		return tx.InsertSession(ctx, record)
	})
}

// sessionIDs lists the stored administrative session identities of one account.
func sessionIDs(t *testing.T, db *gorm.DB, accountID string) []string {
	t.Helper()
	var ids []string
	require.NoError(t, db.Raw(`SELECT id FROM administrative_sessions WHERE account_id = ?`, accountID).Scan(&ids).Error)
	return ids
}

// TestAdministrativeLoginReplacesEarlierSessionsOnPostgres proves replacement, rollback, concurrency and isolation.
func TestAdministrativeLoginReplacesEarlierSessionsOnPostgres(t *testing.T) {
	db := accessDB(t)
	ctx := context.Background()
	store := accessrepository.Store{DB: db}
	now := testdata.Now()

	t.Run("committed login deletes earlier sessions and keeps other accounts", func(t *testing.T) {
		account := replacementAccount(t, store)
		other := replacementAccount(t, store)
		first := newReplacementSession(account.ID, now)
		bystander := newReplacementSession(other.ID, now)
		require.NoError(t, store.Insert(ctx, first))
		require.NoError(t, store.Insert(ctx, bystander))
		second := newReplacementSession(account.ID, now.Add(time.Minute))
		require.NoError(t, loginAs(ctx, store, second))
		assert.Equal(t, []string{second.ID}, sessionIDs(t, db, account.ID))
		_, err := store.FindByTokenHash(ctx, first.TokenHash)
		require.ErrorIs(t, err, accessport.ErrNotFound)
		assert.Equal(t, []string{bystander.ID}, sessionIDs(t, db, other.ID))
	})

	t.Run("rolled back login leaves earlier sessions valid", func(t *testing.T) {
		account := replacementAccount(t, store)
		other := replacementAccount(t, store)
		first := newReplacementSession(account.ID, now)
		taken := newReplacementSession(other.ID, now)
		require.NoError(t, store.Insert(ctx, first))
		require.NoError(t, store.Insert(ctx, taken))
		failed := newReplacementSession(account.ID, now.Add(time.Minute))
		failed.TokenHash = taken.TokenHash
		err := loginAs(ctx, store, failed)
		var conflict accessport.TokenConflictError
		require.ErrorAs(t, err, &conflict)
		assert.Equal(t, []string{first.ID}, sessionIDs(t, db, account.ID))
		got, findErr := store.FindByTokenHash(ctx, first.TokenHash)
		require.NoError(t, findErr)
		assert.Nil(t, got.RevokedAt)
	})

	t.Run("concurrent logins leave exactly one row", func(t *testing.T) {
		account := replacementAccount(t, store)
		require.NoError(t, store.Insert(ctx, newReplacementSession(account.ID, now)))
		const logins = 8
		var wg sync.WaitGroup
		errs := make([]error, logins)
		for i := range logins {
			wg.Add(1)
			go func() {
				defer wg.Done()
				errs[i] = loginAs(ctx, store, newReplacementSession(account.ID, now.Add(time.Duration(i+1)*time.Minute)))
			}()
		}
		wg.Wait()
		for _, err := range errs {
			require.NoError(t, err)
		}
		assert.Len(t, sessionIDs(t, db, account.ID), 1)
	})

	t.Run("logout keeps the row with revoked_at", func(t *testing.T) {
		account := replacementAccount(t, store)
		current := newReplacementSession(account.ID, now)
		require.NoError(t, loginAs(ctx, store, current))
		revoked := now.Add(time.Minute)
		require.NoError(t, store.UpdateRevocation(ctx, current.ID, &revoked))
		got, err := store.FindByTokenHash(ctx, current.TokenHash)
		require.NoError(t, err)
		require.NotNil(t, got.RevokedAt)
		assert.True(t, revoked.Equal(*got.RevokedAt))
	})
}
