package integration_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/rfanazhari/amer-prototype/application/repository/accessport"
	"github.com/rfanazhari/amer-prototype/application/usecase/accessusecase"
	"github.com/rfanazhari/amer-prototype/infrastructure/repository/accessrepository"
	"github.com/rfanazhari/amer-prototype/test/testdata"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// fixedClock supplies one trusted instant for retention boundary proofs.
type fixedClock struct{ at time.Time }

// Now returns the configured instant.
func (c fixedClock) Now() time.Time { return c.at }

// retentionSession describes one seeded session relative to the run instant.
type retentionSession struct {
	name      string
	expiresIn time.Duration
	revokedIn *time.Duration
	wantGone  bool
}

// seedRetentionSession stores one session for the account with expiry and revocation offsets from now.
func seedRetentionSession(t *testing.T, store accessrepository.Store, accountID string, now time.Time, spec retentionSession) string {
	t.Helper()
	ctx := context.Background()
	digest := sha256.Sum256([]byte(uuid.NewString()))
	record := accessport.SessionRecord{
		ID:        uuid.NewString(),
		AccountID: accountID,
		TokenHash: hex.EncodeToString(digest[:]),
		CreatedAt: now.Add(spec.expiresIn - 100*24*time.Hour),
		ExpiresAt: now.Add(spec.expiresIn),
	}
	require.NoError(t, store.Insert(ctx, record), spec.name)
	if spec.revokedIn != nil {
		revoked := now.Add(*spec.revokedIn)
		require.NoError(t, store.UpdateRevocation(ctx, record.ID, &revoked), spec.name)
	}
	return record.ID
}

// countRows returns the row count of an existing table, or -1 when the table is not installed.
func countRows(t *testing.T, db *gorm.DB, table string) int64 {
	t.Helper()
	var exists bool
	require.NoError(t, db.Raw(`SELECT to_regclass(?) IS NOT NULL`, "public."+table).Scan(&exists).Error)
	if !exists {
		return -1
	}
	var count int64
	require.NoError(t, db.Raw(`SELECT COUNT(*) FROM `+table).Scan(&count).Error)
	return count
}

// TestAdministrativeSessionRetentionOnPostgres proves the age boundary, batch limit, rerun and untouched tables.
func TestAdministrativeSessionRetentionOnPostgres(t *testing.T) {
	db := accessDB(t)
	ctx := context.Background()
	store := accessrepository.Store{DB: db}
	require.NoError(t, db.Exec(`DELETE FROM administrative_sessions`).Error)
	account := testdata.AdministrativeAccountRecord(t)
	account.ID = uuid.NewString()
	account.Email = "retention-" + uuid.NewString() + "@example.com"
	require.NoError(t, store.WithinAccounts(ctx, func(tx accessport.AccountTx) error { return tx.Insert(ctx, account) }))
	now := testdata.Now().Add(365 * 24 * time.Hour)
	week := 7 * 24 * time.Hour
	revokedLong := -8 * 24 * time.Hour
	revokedExact := -week
	revokedRecent := -time.Hour
	specs := []retentionSession{
		{name: "expired one millisecond past grace", expiresIn: -week - time.Millisecond, wantGone: true},
		{name: "expired exactly at grace", expiresIn: -week},
		{name: "expired inside grace", expiresIn: -6 * 24 * time.Hour},
		{name: "expires exactly now", expiresIn: 0},
		{name: "still valid", expiresIn: time.Hour},
		{name: "revoked long ago while unexpired", expiresIn: time.Hour, revokedIn: &revokedLong, wantGone: true},
		{name: "revoked exactly at grace", expiresIn: time.Hour, revokedIn: &revokedExact},
		{name: "revoked recently", expiresIn: time.Hour, revokedIn: &revokedRecent},
	}
	ids := make([]string, len(specs))
	for i, spec := range specs {
		ids[i] = seedRetentionSession(t, store, account.ID, now, spec)
	}
	untouched := map[string]int64{}
	for _, table := range []string{"administrative_accounts", "interview_invitations", "candidate_sessions", "candidates"} {
		untouched[table] = countRows(t, db, table)
	}
	var canDelete bool
	require.NoError(t, db.Raw(`SELECT has_table_privilege(current_user, 'public.administrative_sessions', 'DELETE')`).Scan(&canDelete).Error)
	require.True(t, canDelete)

	prune := accessusecase.PruneSessions{Retention: store, Clock: fixedClock{at: now}}
	removed, err := prune.Execute(ctx)
	require.NoError(t, err)
	assert.Equal(t, int64(2), removed)
	for i, spec := range specs {
		var present int64
		require.NoError(t, db.Raw(`SELECT COUNT(*) FROM administrative_sessions WHERE id = ?`, ids[i]).Scan(&present).Error)
		assert.Equal(t, map[bool]int64{true: 0, false: 1}[spec.wantGone], present, spec.name)
	}
	again, err := prune.Execute(ctx)
	require.NoError(t, err)
	assert.Zero(t, again)
	for table, before := range untouched {
		assert.Equal(t, before, countRows(t, db, table), table)
	}

	t.Run("batch limit bounds one statement", func(t *testing.T) {
		for i := 0; i < 3; i++ {
			seedRetentionSession(t, store, account.ID, now, retentionSession{name: "stale", expiresIn: -8 * 24 * time.Hour})
		}
		first, err := store.DeleteStale(ctx, now.Add(-week), 2)
		require.NoError(t, err)
		assert.Equal(t, int64(2), first)
		second, err := store.DeleteStale(ctx, now.Add(-week), 2)
		require.NoError(t, err)
		assert.Equal(t, int64(1), second)
		third, err := store.DeleteStale(ctx, now.Add(-week), 2)
		require.NoError(t, err)
		assert.Zero(t, third)
	})
}
