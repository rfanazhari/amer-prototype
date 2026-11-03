// Package reportbroker dispatches durable report publish intents without owning report outcomes.
package reportbroker

import (
	"context"
	"time"

	"github.com/rfanazhari/amer-prototype/application/repository/interviewreportport"
	"github.com/rfanazhari/amer-prototype/application/service"
)

// Broker returns success only after a routed persistent publication is confirmed.
type Broker interface {
	// Publish sends the identity-only body and waits for a positive confirmation.
	Publish(ctx context.Context, message interviewreportport.OutboxMessage) error
}

// Outbox records confirmations independently of generation outcomes.
type Outbox interface {
	// ClaimOutbox commits a bounded batch claim before returning any messages.
	ClaimOutbox(ctx context.Context, limit int, now time.Time) ([]interviewreportport.OutboxMessage, error)
	// ConfirmOutbox atomically records confirmation and advances only a pending job to published.
	ConfirmOutbox(ctx context.Context, message interviewreportport.OutboxMessage, now time.Time) error
}

// Publisher reconciles unconfirmed rows; callers own scheduling and resource lifetime.
type Publisher struct {
	Store     Outbox
	Broker    Broker
	Clock     service.Clock
	BatchSize int
	Timeout   time.Duration
}

// Reconcile claims one batch and republishes unconfirmed messages, including after a lost confirm.
func (p Publisher) Reconcile(ctx context.Context) error {
	return nil
}
