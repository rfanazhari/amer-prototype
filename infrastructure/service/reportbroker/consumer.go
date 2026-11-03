package reportbroker

import (
	"context"

	amqp "github.com/rabbitmq/amqp091-go"
)

// JobExecutor processes one durable job and returns database failures to the consumer.
type JobExecutor interface {
	// Execute finishes processing or reports why this delivery must remain unacknowledged.
	Execute(context.Context, string) error
}

// Consumer handles manual-ack deliveries; its caller owns the channel and stop policy.
// It is deliberately not registered in the runtime before T15.
type Consumer struct {
	Worker JobExecutor
}

// Handle acknowledges only successfully processed, well-formed report identities.
// On error the caller must stop consumption and close the channel before recovery.
func (c Consumer) Handle(ctx context.Context, delivery amqp.Delivery) error {
	return nil
}
