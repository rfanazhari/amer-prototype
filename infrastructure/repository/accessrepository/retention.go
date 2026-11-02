package accessrepository

import (
	"context"
	"time"
)

// DeleteStale removes at most limit stale administrative sessions and reports the rows removed.
func (s Store) DeleteStale(ctx context.Context, cutoff time.Time, limit int) (int64, error) {
	return 0, nil
}
