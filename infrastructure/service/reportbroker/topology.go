package reportbroker

import (
	"errors"

	amqp "github.com/rabbitmq/amqp091-go"
)

// TopologyChannel declares the report isolation resources on a caller-owned channel.
type TopologyChannel interface {
	// ExchangeDeclare creates the durable direct dead-letter exchange.
	ExchangeDeclare(string, string, bool, bool, bool, bool, amqp.Table) error
	// QueueDeclare creates a durable queue with its delivery guarantees.
	QueueDeclare(string, bool, bool, bool, bool, amqp.Table) (amqp.Queue, error)
	// QueueBind routes malformed messages into quarantine.
	QueueBind(string, string, string, bool, amqp.Table) error
}

// Topology provisions isolation before a caller starts report intake; runtime wiring remains T15.
type Topology struct {
	Channel TopologyChannel
}

// Declare prepares quarantine and the named source queue, failing closed on any broker error.
func (t Topology) Declare(source string) error {
	if t.Channel == nil || source == "" || source == "interview-report.invalid" {
		return errors.New("report topology requires a channel and a distinct source queue")
	}
	if err := t.Channel.ExchangeDeclare("interview-report.dlx", "direct", true, false, false, false, nil); err != nil {
		return err
	}
	if _, err := t.Channel.QueueDeclare("interview-report.invalid", true, false, false, false,
		amqp.Table{"x-queue-type": "quorum"}); err != nil {
		return err
	}
	if err := t.Channel.QueueBind("interview-report.invalid", "malformed", "interview-report.dlx", false, nil); err != nil {
		return err
	}
	_, err := t.Channel.QueueDeclare(source, true, false, false, false, amqp.Table{
		"x-queue-type":              "quorum",
		"x-dead-letter-exchange":    "interview-report.dlx",
		"x-dead-letter-routing-key": "malformed",
		"x-dead-letter-strategy":    "at-least-once",
		"x-overflow":                "reject-publish",
		// PostgreSQL owns valid-job recovery; broker redelivery must not quarantine database outages.
		"x-delivery-limit": -1,
	})
	return err
}

var _ TopologyChannel = (*amqp.Channel)(nil)
