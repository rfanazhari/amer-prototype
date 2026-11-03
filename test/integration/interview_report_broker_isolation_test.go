package integration_test

import (
	"testing"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/rfanazhari/amer-prototype/infrastructure/service/reportbroker"
	"github.com/rfanazhari/amer-prototype/test/mocks/reportbrokermocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// isolationCase selects a malformed body that must bypass all application and SQL work.
type isolationCase struct {
	name string
	body string
}

// TestInterviewReportRabbitMQIsolation proves real dead-letter routing, no requeue and interview isolation.
func TestInterviewReportRabbitMQIsolation(t *testing.T) {
	tests := []isolationCase{
		{name: "invalid JSON", body: `{`},
		{name: "missing identity", body: `{}`},
		{name: "mismatched identity", body: `{"jobKind":"interview_report","jobId":"job","messageId":"other"}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := newReportBrokerFixture(t)
			f := newReportFixture(t)
			id, now := admittedWorkerJob(t, f)
			assignmentBefore, executionBefore := f.interviewRows(t, f.db)
			quarantine, err := b.connection.Channel()
			require.NoError(t, err)
			defer func() { _ = quarantine.Close() }()
			isolated, err := quarantine.ConsumeWithContext(b.ctx, "interview-report.invalid", "", false, false, false, false, nil)
			require.NoError(t, err)
			// This is an intentionally malformed transport body, sent to the actual source queue.
			require.NoError(t, b.channel.PublishWithContext(b.ctx, "", b.queue, true, false,
				amqp.Publishing{MessageId: "malformed", DeliveryMode: amqp.Persistent, Body: []byte(tt.body)}))
			channel, deliveries := b.consume(t)
			delivery := b.receive(t, deliveries)
			consumer := reportbroker.Consumer{Worker: reportbrokermocks.NewJobExecutor(t)}
			require.NoError(t, consumer.Handle(b.ctx, delivery))
			dead := b.receive(t, isolated)
			assert.Equal(t, delivery.Body, dead.Body)
			assert.Equal(t, delivery.MessageId, dead.MessageId)
			assert.Equal(t, "interview-report.dlx", dead.Exchange)
			assert.Equal(t, "malformed", dead.RoutingKey)
			assert.Equal(t, "rejected", dead.Headers["x-first-death-reason"])
			require.NoError(t, dead.Ack(false))
			b.assertBrokerDrained(t, channel)
			assignmentAfter, executionAfter := f.interviewRows(t, f.db)
			assert.Equal(t, assignmentBefore, assignmentAfter)
			assert.Equal(t, executionBefore, executionAfter)
			assert.Zero(t, f.count(t, `SELECT count(*) FROM interview_report_attempts WHERE job_id=?`, id))
			// A valid message behind the malformed delivery still reaches the normal worker.
			require.NoError(t, b.client.Publish(b.ctx, f.message(t, id)))
			channel, deliveries = b.consume(t)
			worker := reportBrokerWorker(t, f, now)
			// Existing T11 cases prove full provider processing; an active job proves safe duplicate handling here.
			_, err = f.reports.ClaimAttempt(b.ctx, id, true, now)
			require.NoError(t, err)
			require.NoError(t, (reportbroker.Consumer{Worker: worker}).Handle(b.ctx, b.receive(t, deliveries)))
			b.assertBrokerDrained(t, channel)
			empty, err := quarantine.QueueDeclarePassive("interview-report.invalid", true, false, false, false, nil)
			require.NoError(t, err)
			assert.Zero(t, empty.Messages)
		})
	}
}
