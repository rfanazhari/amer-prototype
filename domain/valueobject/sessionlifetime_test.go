package valueobject_test

import (
	"testing"
	"time"

	"github.com/rfanazhari/amer-prototype/domain/valueobject"
	"github.com/rfanazhari/amer-prototype/test/testdata"
	"github.com/stretchr/testify/require"
)

// lifetimeCreateCase defines creation inputs for an absolute session lifetime.
type lifetimeCreateCase struct {
	name      string
	createdAt time.Time
	duration  time.Duration
	wantErr   bool
}

// TestNewSessionLifetimeRejectsInvalidCreation checks zero time, non-positive duration and exact administrative span.
func TestNewSessionLifetimeRejectsInvalidCreation(t *testing.T) {
	now := testdata.Now()
	tests := []lifetimeCreateCase{
		{"administrative duration", now, valueobject.AdministrativeDuration, false},
		{"zero created", time.Time{}, valueobject.AdministrativeDuration, true},
		{"zero duration", now, 0, true},
		{"negative duration", now, -time.Millisecond, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := valueobject.NewSessionLifetime(tt.createdAt, tt.duration)
			if tt.wantErr {
				require.ErrorIs(t, err, valueobject.ErrInvalidSessionLifetime)
				require.False(t, got.Valid())
				return
			}
			require.NoError(t, err)
			require.True(t, got.Valid())
			require.Equal(t, tt.createdAt, got.CreatedAt())
			require.Equal(t, tt.createdAt.Add(tt.duration), got.ExpiresAt())
		})
	}
}

// lifetimeRestoreCase defines stored lifetime facts expected to restore or fail.
type lifetimeRestoreCase struct {
	name      string
	createdAt time.Time
	expiresAt time.Time
	wantErr   bool
}

// TestRestoreSessionLifetimePreservesStoredDeadline restores historical expiry without recomputing duration.
func TestRestoreSessionLifetimePreservesStoredDeadline(t *testing.T) {
	now := testdata.Now()
	tests := []lifetimeRestoreCase{
		{"stored administrative expiry", now, now.Add(valueobject.AdministrativeDuration), false},
		{"historical nonstandard span", now, now.Add(90 * time.Minute), false},
		{"zero created", time.Time{}, now.Add(time.Hour), true},
		{"zero expiry", now, time.Time{}, true},
		{"expiry equal created", now, now, true},
		{"expiry before created", now, now.Add(-time.Millisecond), true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := valueobject.RestoreSessionLifetime(tt.createdAt, tt.expiresAt)
			if tt.wantErr {
				require.ErrorIs(t, err, valueobject.ErrInvalidSessionLifetime)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.createdAt, got.CreatedAt())
			require.Equal(t, tt.expiresAt, got.ExpiresAt())
		})
	}
}

// currentCase defines exact expiry-boundary observations for a current lifetime.
type currentCase struct {
	name    string
	offset  time.Duration
	current bool
}

// TestSessionLifetimeCurrentUsesExactExpiryBoundary treats equality with expiry as no longer current.
func TestSessionLifetimeCurrentUsesExactExpiryBoundary(t *testing.T) {
	now := testdata.Now()
	lifetime, err := valueobject.NewSessionLifetime(now, valueobject.AdministrativeDuration)
	require.NoError(t, err)
	tests := []currentCase{
		{"at creation", 0, true},
		{"one millisecond before expiry", valueobject.AdministrativeDuration - time.Millisecond, true},
		{"at expiry", valueobject.AdministrativeDuration, false},
		{"after expiry", valueobject.AdministrativeDuration + time.Millisecond, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.current, lifetime.Current(now.Add(tt.offset)))
		})
	}
}
