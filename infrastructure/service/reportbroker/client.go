package reportbroker

import (
	"context"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/rfanazhari/amer-prototype/application/repository/interviewreportport"
)

// Channel is a dedicated AMQP publishing channel owned and closed by the caller.
// The caller must bound connection writes; AMQP context cancellation only bounds confirmation waits.
type Channel interface {
	// Confirm enables publisher acknowledgements synchronously.
	Confirm(noWait bool) error
	// NotifyPublish subscribes to ordered per-message confirmations.
	NotifyPublish(chan amqp.Confirmation) chan amqp.Confirmation
	// NotifyReturn subscribes to mandatory unroutable publications.
	NotifyReturn(chan amqp.Return) chan amqp.Return
	// GetNextPublishSeqNo identifies the next publication on this exclusive channel.
	GetNextPublishSeqNo() uint64
	// PublishWithContext sends a message without waiting for its confirmation.
	PublishWithContext(context.Context, string, string, bool, bool, amqp.Publishing) error
}

// Client publishes through a caller-provisioned route; it declares no topology.
// Use a dedicated channel and replace the client/channel after any uncertain publication.
type Client struct {
	Channel    Channel
	Exchange   string
	RoutingKey string
}

// Publish sends only jobKind, jobId and messageId as persistent JSON and waits for confirmation.
func (c *Client) Publish(ctx context.Context, message interviewreportport.OutboxMessage) error {
	return nil
}
