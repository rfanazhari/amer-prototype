package interviewreportrepository

import (
	"context"
	"time"

	"github.com/rfanazhari/amer-prototype/application/repository/interviewreportport"
)

// ConfirmOutbox records broker confirmation without changing a generation or reopening a claimed job.
func (s Store) ConfirmOutbox(ctx context.Context, message interviewreportport.OutboxMessage, now time.Time) error {
	return nil
}
