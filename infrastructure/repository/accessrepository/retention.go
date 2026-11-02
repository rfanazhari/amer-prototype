package accessrepository

import (
	"context"
	"errors"
	"time"
)

// staleSessionsDelete removes one bounded batch; only administrative_sessions is named and rows locked elsewhere are skipped.
const staleSessionsDelete = `DELETE FROM administrative_sessions WHERE id IN (SELECT id FROM administrative_sessions WHERE expires_at < ? OR revoked_at < ? ORDER BY id LIMIT ? FOR UPDATE SKIP LOCKED)`

// DeleteStale removes at most limit stale administrative sessions and reports the rows removed.
func (s Store) DeleteStale(ctx context.Context, cutoff time.Time, limit int) (int64, error) {
	if limit <= 0 {
		return 0, errors.New("session cleanup batch limit must be positive")
	}
	result := s.DB.WithContext(ctx).Exec(staleSessionsDelete, cutoff, cutoff, limit)
	return result.RowsAffected, result.Error
}
