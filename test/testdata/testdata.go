package testdata

import (
	"github.com/rfanazhari/amer-prototype/application/repository/accessport"
	"github.com/rfanazhari/amer-prototype/domain/aggregate"
	"github.com/rfanazhari/amer-prototype/domain/enum"
	"github.com/rfanazhari/amer-prototype/domain/valueobject"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

const ID = "9d96c891-8118-46d4-92d2-79da2337faba"

// SessionID is a distinct administrative session identity used by access persistence tests.
const SessionID = "7c3e2a11-4b0d-4c8a-9f21-1a2b3c4d5e6f"

// TokenHash is a 64-character lowercase SHA-256 hex digest used as a stored session credential.
const TokenHash = "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"

// PasswordHash is a nonempty encoded password digest that is not a session token namespace.
const PasswordHash = "$argon2id$v=19$m=19456,t=2,p=1$c2FsdHNhbHRzYWx0c2FsdA$c3RvcmVkLWhhc2g"

// Now returns the fixed UTC baseline used by public domain and application scenarios.
func Now() time.Time { return time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC) }

// Task returns a fresh valid aggregate in the requested lifecycle state, created through public domain operations.
func Task(t *testing.T, state enum.TaskState, due *time.Time) *aggregate.Task {
	t.Helper()
	title, err := valueobject.NewTitle("Prepare report")
	require.NoError(t, err)
	task, err := aggregate.NewTask(ID, title, due, Now())
	require.NoError(t, err)
	if state == enum.InProgress || state == enum.Done {
		require.NoError(t, task.Start())
	}
	if state == enum.Done {
		_, err = task.Complete(Now().Add(time.Minute))
		require.NoError(t, err)
	}
	return task
}

// AdministrativeAccountRecord returns a fresh ADMIN identity with canonical email and hashed password.
func AdministrativeAccountRecord(t *testing.T) accessport.AccountRecord {
	t.Helper()
	return accessport.AccountRecord{
		ID:           ID,
		Name:         "Sinta Wijaya",
		Email:        "sinta@example.com",
		PasswordHash: PasswordHash,
		Role:         enum.Admin,
		Status:       enum.Active,
		CreatedAt:    Now(),
		UpdatedAt:    Now(),
	}
}

// AdministrativeSessionRecord returns a fresh unrevoked session bound to the administrative fixture account.
func AdministrativeSessionRecord(t *testing.T) accessport.SessionRecord {
	t.Helper()
	return accessport.SessionRecord{
		ID:        SessionID,
		AccountID: ID,
		TokenHash: TokenHash,
		CreatedAt: Now(),
		ExpiresAt: Now().Add(2 * time.Hour),
	}
}
