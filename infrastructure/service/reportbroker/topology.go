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
	return errors.New("report isolation is not implemented")
}
