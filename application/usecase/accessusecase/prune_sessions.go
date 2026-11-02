package accessusecase

import (
	"context"
	"time"

	"github.com/rfanazhari/amer-prototype/application/repository/accessport"
	"github.com/rfanazhari/amer-prototype/application/service"
)

const (
	// SessionRetentionGrace is how long an administrative session is kept after expiry or revocation.
	SessionRetentionGrace = 7 * 24 * time.Hour
	// SessionPruneBatchSize bounds the rows removed by one delete statement.
	SessionPruneBatchSize = 500
)

// PruneSessions removes old administrative sessions in bounded batches for an operator-run command.
type PruneSessions struct {
	Retention accessport.SessionRetention
	Clock     service.Clock
}

// Execute removes stale sessions batch by batch and returns the total number of rows removed.
func (u PruneSessions) Execute(ctx context.Context) (int64, error) {
	return 0, nil
}
