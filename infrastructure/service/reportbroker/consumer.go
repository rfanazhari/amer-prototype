package reportbroker

import (
	"context"
	"encoding/json"
	"errors"

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
// Malformed deliveries are rejected into the predeclared quarantine; nil means settled, not necessarily ACKed.
// On error the caller must stop consumption and close the channel before recovery.
func (c Consumer) Handle(ctx context.Context, delivery amqp.Delivery) error {
	if c.Worker == nil {
		return errors.New("report consumer requires a worker")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	var message messageBody
	if err := json.Unmarshal(delivery.Body, &message); err != nil {
		return delivery.Reject(false)
	}
	if message.JobKind != "interview_report" || message.JobID == "" || message.MessageID == "" || message.MessageID != delivery.MessageId {
		return delivery.Reject(false)
	}
	if err := c.Worker.Execute(ctx, message.JobID); err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	return delivery.Ack(false)
}
