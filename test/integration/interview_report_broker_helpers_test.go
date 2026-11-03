package integration_test

import (
	"context"
	"net"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/rfanazhari/amer-prototype/application/repository/interviewreportport"
	"github.com/rfanazhari/amer-prototype/application/usecase/interviewreportusecase"
	"github.com/rfanazhari/amer-prototype/infrastructure/service/reportbroker"
	"github.com/rfanazhari/amer-prototype/test/mocks"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// reportBrokerFixture owns a dedicated broker connection, queue and publisher for one proof.
type reportBrokerFixture struct {
	connection *amqp.Connection
	channel    *amqp.Channel
	queue      string
	client     *reportbroker.Client
	ctx        context.Context
}

// newReportBrokerFixture uses an environment-only URL and never emits credential-bearing dial errors.
func newReportBrokerFixture(t *testing.T) reportBrokerFixture {
	t.Helper()
	address := os.Getenv("TEST_RABBITMQ_URL")
	if address == "" {
		t.Skip("TEST_RABBITMQ_URL is required; real RabbitMQ proof skipped")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	t.Cleanup(cancel)
	connection, err := amqp.DialConfig(address, amqp.Config{Dial: amqp.DefaultDial(5 * time.Second), Heartbeat: time.Second})
	if err != nil {
		t.Fatal("RabbitMQ connection failed; URL and cause withheld to protect credentials")
	}
	t.Cleanup(func() { _ = connection.CloseDeadline(time.Now().Add(time.Second)) })
	channel, err := connection.Channel()
	require.NoError(t, err)
	queue := "irp-proof-" + uuid.NewString()
	require.NoError(t, (reportbroker.Topology{Channel: channel}).Declare(queue))
	t.Cleanup(func() {
		if !connection.IsClosed() {
			cleanup, err := connection.Channel()
			require.NoError(t, err)
			_, err = cleanup.QueueDelete(queue, false, false, false)
			require.NoError(t, err)
			require.NoError(t, cleanup.Close())
		}
	})
	return reportBrokerFixture{connection: connection, channel: channel, queue: queue,
		client: &reportbroker.Client{Channel: channel, RoutingKey: queue}, ctx: ctx}
}

// consume starts a manual-ack, prefetch-one consumer on a separately owned connection.
func (b reportBrokerFixture) consume(t *testing.T) (*amqp.Channel, <-chan amqp.Delivery) {
	t.Helper()
	connection, err := amqp.DialConfig(os.Getenv("TEST_RABBITMQ_URL"), amqp.Config{
		Dial: func(network, address string) (net.Conn, error) {
			return (&net.Dialer{Timeout: 5 * time.Second}).DialContext(b.ctx, network, address)
		},
	})
	if err != nil {
		t.Fatal("RabbitMQ consumer connection failed; URL and cause withheld")
	}
	t.Cleanup(func() { _ = connection.CloseDeadline(time.Now().Add(time.Second)) })
	channel, err := connection.Channel()
	require.NoError(t, err)
	require.NoError(t, channel.Qos(1, 0, false))
	deliveries, err := channel.ConsumeWithContext(b.ctx, b.queue, "", false, false, false, false, nil)
	require.NoError(t, err)
	return channel, deliveries
}

// receive waits for real broker delivery with a bounded deadline.
func (b reportBrokerFixture) receive(t *testing.T, deliveries <-chan amqp.Delivery) amqp.Delivery {
	t.Helper()
	select {
	case delivery, open := <-deliveries:
		require.True(t, open, "consumer closed before delivery")
		return delivery
	case <-b.ctx.Done():
		t.Fatal("timed out waiting for RabbitMQ delivery")
		return amqp.Delivery{}
	}
}

// message reads a committed outbox identity without substituting SQL expectations.
func (f reportFixture) message(t *testing.T, jobID string) interviewreportport.OutboxMessage {
	t.Helper()
	var id string
	require.NoError(t, f.db.Table("outbox_messages").Select("id").Where("job_id = ?", jobID).Scan(&id).Error)
	require.NotEmpty(t, id)
	return interviewreportport.OutboxMessage{ID: id, JobID: jobID, JobKind: "interview_report"}
}

// reportBrokerWorker injects finalized evidence and a controlled clock, retaining the real PostgreSQL store.
func reportBrokerWorker(t *testing.T, f reportFixture, now time.Time) interviewreportusecase.Worker {
	t.Helper()
	evidence := mocks.NewInterviewReportEvidence(t)
	evidence.On("Finalized", mock.Anything, f.executionID).Return([]interviewreportport.EvidenceEntry{{ID: uuid.NewString(), Revision: 1}}, nil)
	clock := mocks.NewClock(t)
	clock.On("Now").Return(now)
	return interviewreportusecase.Worker{Store: f.reports, Evidence: evidence, Writer: mocks.NewInterviewReportWriter(t), Clock: clock}
}

// assertBrokerDrained synchronizes ack on the consumer channel, then inspects after closing it.
func (b reportBrokerFixture) assertBrokerDrained(t *testing.T, channel *amqp.Channel) {
	t.Helper()
	require.NoError(t, channel.Qos(1, 0, false))
	require.NoError(t, channel.Close())
	queue, err := b.channel.QueueDeclarePassive(b.queue, true, false, false, false, nil)
	require.NoError(t, err)
	require.Zero(t, queue.Messages, "successful handling must ack rather than requeue on stop")
}
