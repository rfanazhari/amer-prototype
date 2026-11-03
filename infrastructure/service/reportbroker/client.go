package reportbroker

import (
	"context"
	"encoding/json"
	"errors"
	"sync"

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
	mu         sync.Mutex
	confirms   chan amqp.Confirmation
	returns    chan amqp.Return
	failed     bool
}

// messageBody deliberately excludes report content and all stored outbox metadata.
type messageBody struct {
	JobKind   string `json:"jobKind"`
	JobID     string `json:"jobId"`
	MessageID string `json:"messageId"`
}

var (
	errUncertain  = errors.New("report publication unconfirmed; replace the dedicated channel and client")
	errUnroutable = errors.New("report publication was returned as unroutable")
	errRejected   = errors.New("report publication was not acknowledged")
)

var _ Channel = (*amqp.Channel)(nil)

// waitConfirm accepts only this publication's positive acknowledgement and checks mandatory returns.
func (c *Client) waitConfirm(ctx context.Context, sequence uint64) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-c.returns:
		return errUnroutable
	case confirm, open := <-c.confirms:
		if !open || confirm.DeliveryTag != sequence || !confirm.Ack {
			return errRejected
		}
		// RabbitMQ sends basic.return before basic.ack for an unroutable mandatory publish.
		select {
		case <-c.returns:
			return errUnroutable
		default:
			return nil
		}
	}
}

// Publish sends only jobKind, jobId and messageId as persistent JSON and waits for confirmation.
func (c *Client) Publish(ctx context.Context, message interviewreportport.OutboxMessage) (err error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if err := ctx.Err(); err != nil {
		return err
	}
	if c.Channel == nil || c.RoutingKey == "" || message.JobKind != "interview_report" || message.JobID == "" || message.ID == "" {
		return errors.New("invalid report publication dependencies, route or identity")
	}
	if c.failed {
		return errUncertain
	}
	defer func() { c.failed = err != nil }()
	if c.confirms == nil {
		if err := c.Channel.Confirm(false); err != nil {
			return err
		}
		c.confirms = c.Channel.NotifyPublish(make(chan amqp.Confirmation, 1))
		c.returns = c.Channel.NotifyReturn(make(chan amqp.Return, 1))
	}
	body, err := json.Marshal(messageBody{JobKind: message.JobKind, JobID: message.JobID, MessageID: message.ID})
	if err != nil {
		return err
	}
	sequence := c.Channel.GetNextPublishSeqNo()
	if err := c.Channel.PublishWithContext(ctx, c.Exchange, c.RoutingKey, true, false, amqp.Publishing{
		ContentType: "application/json", DeliveryMode: amqp.Persistent, MessageId: message.ID, Body: body,
	}); err != nil {
		return err
	}
	return c.waitConfirm(ctx, sequence)
}
