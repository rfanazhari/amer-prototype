package accessport

import (
	"context"
	"time"
)

// SessionRetention removes administrative sessions that can no longer authorize and are past their retention grace.
type SessionRetention interface {
	// DeleteStale removes at most limit sessions whose expires_at or revoked_at is strictly before cutoff and returns how many rows were removed.
	DeleteStale(ctx context.Context, cutoff time.Time, limit int) (int64, error)
}
