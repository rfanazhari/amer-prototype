package integration_test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/rfanazhari/amer-prototype/application/usecase/interviewreportusecase"
	"github.com/rfanazhari/amer-prototype/domain/aggregate/interviewreport"
	"github.com/rfanazhari/amer-prototype/infrastructure/repository/interviewreportrepository"
	"github.com/rfanazhari/amer-prototype/infrastructure/service/clock"
	"github.com/rfanazhari/amer-prototype/infrastructure/service/reportbroker"
	"github.com/rfanazhari/amer-prototype/test/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// reportBrokerCase selects a proof crossing actual RabbitMQ and PostgreSQL boundaries.
type reportBrokerCase struct {
	name   string
	verify func(*testing.T, reportFixture, reportBrokerFixture)
}

// TestInterviewReportRabbitMQ proves well-formed delivery semantics without sqlmock or broker mocks.
func TestInterviewReportRabbitMQ(t *testing.T) {
	tests := []reportBrokerCase{
		{name: "persistent confirm records committed outbox", verify: proveReportConfirm},
		{name: "consumer stops before ack and broker redelivers", verify: proveReportRedelivery},
		{name: "active duplicate acks without another attempt", verify: proveReportActiveDuplicate},
		{name: "terminal duplicate acks without another attempt", verify: proveReportTerminalDuplicate},
		{name: "database error remains unacknowledged", verify: proveReportDatabaseError},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			broker := newReportBrokerFixture(t)
			tt.verify(t, newReportFixture(t), broker)
		})
	}
}

// proveReportConfirm observes a real publisher confirm, payload and durable confirmation timestamp.
func proveReportConfirm(t *testing.T, f reportFixture, b reportBrokerFixture) {
	id, _ := admittedWorkerJob(t, f)
	message := f.message(t, id)
	publisher := reportbroker.Publisher{Store: f.reports, Broker: b.client, Clock: clock.System{}, BatchSize: 1, Timeout: 5 * time.Second}
	require.NoError(t, publisher.Reconcile(b.ctx))
	assert.EqualValues(t, 1, f.count(t, `SELECT count(*) FROM outbox_messages WHERE id=? AND confirmed_at IS NOT NULL`, message.ID))
	assert.EqualValues(t, 1, f.count(t, `SELECT count(*) FROM interview_report_jobs WHERE id=? AND state='published' AND attempt_count=0`, id))
	channel, deliveries := b.consume(t)
	delivery := b.receive(t, deliveries)
	assert.Equal(t, uint8(amqp.Persistent), delivery.DeliveryMode)
	assert.Equal(t, "application/json", delivery.ContentType)
	assert.Equal(t, message.ID, delivery.MessageId)
	var body map[string]string
	require.NoError(t, json.Unmarshal(delivery.Body, &body))
	assert.Equal(t, map[string]string{"jobKind": "interview_report", "jobId": id, "messageId": message.ID}, body)
	require.NoError(t, delivery.Ack(false))
	b.assertBrokerDrained(t, channel)
	assert.EqualValues(t, 1, f.count(t, `SELECT count(*) FROM interview_report_generations WHERE state='pending' AND id IN (SELECT generation_id FROM interview_report_jobs WHERE id=?)`, id))
}

// proveReportRedelivery closes a live consumer before ack and receives the identical persistent message again.
func proveReportRedelivery(t *testing.T, f reportFixture, b reportBrokerFixture) {
	id, _ := admittedWorkerJob(t, f)
	message := f.message(t, id)
	require.NoError(t, b.client.Publish(b.ctx, message))
	channel, deliveries := b.consume(t)
	first := b.receive(t, deliveries)
	require.False(t, first.Redelivered)
	require.NoError(t, channel.Close())
	channel, deliveries = b.consume(t)
	second := b.receive(t, deliveries)
	assert.True(t, second.Redelivered)
	assert.Equal(t, first.Body, second.Body)
	assert.Equal(t, first.MessageId, second.MessageId)
	require.NoError(t, second.Ack(false))
	b.assertBrokerDrained(t, channel)
}

// proveReportActiveDuplicate models a crash after claim but before ack, then processes its redelivery.
func proveReportActiveDuplicate(t *testing.T, f reportFixture, b reportBrokerFixture) {
	id, now := admittedWorkerJob(t, f)
	require.NoError(t, b.client.Publish(b.ctx, f.message(t, id)))
	channel, deliveries := b.consume(t)
	_ = b.receive(t, deliveries)
	claim, err := f.reports.ClaimAttempt(b.ctx, id, true, now)
	require.NoError(t, err)
	require.NoError(t, channel.Close())
	channel, deliveries = b.consume(t)
	delivery := b.receive(t, deliveries)
	require.True(t, delivery.Redelivered)
	consumer := reportbroker.Consumer{Worker: reportBrokerWorker(t, f, now)}
	require.NoError(t, consumer.Handle(b.ctx, delivery))
	b.assertBrokerDrained(t, channel)
	assert.EqualValues(t, 1, f.count(t, `SELECT count(*) FROM interview_report_attempts WHERE job_id=?`, id))
	assert.EqualValues(t, 1, f.count(t, `SELECT count(*) FROM interview_report_jobs WHERE id=? AND state='claimed' AND fencing_token=? AND attempt_count=1`, id, claim.FencingToken))
}

// proveReportTerminalDuplicate processes a job then republishes the same message without another writer call.
func proveReportTerminalDuplicate(t *testing.T, f reportFixture, b reportBrokerFixture) {
	id, now := admittedWorkerJob(t, f)
	worker := reportBrokerWorker(t, f, now)
	worker.Writer.(*mocks.InterviewReportWriter).On("Write", mock.Anything, mock.Anything).
		Return([]interviewreport.StatementInput{{Kind: interviewreport.KindGap, Body: "No further detail recorded."}}, nil).Once()
	consumer := reportbroker.Consumer{Worker: worker}
	message := f.message(t, id)
	for range 2 {
		require.NoError(t, b.client.Publish(b.ctx, message))
		channel, deliveries := b.consume(t)
		require.NoError(t, consumer.Handle(b.ctx, b.receive(t, deliveries)))
		b.assertBrokerDrained(t, channel)
	}
	assert.EqualValues(t, 1, f.count(t, `SELECT count(*) FROM interview_report_attempts WHERE job_id=?`, id))
	assert.EqualValues(t, 1, f.count(t, `SELECT count(*) FROM interview_report_jobs WHERE id=? AND state='succeeded' AND attempt_count=1`, id))
}

// proveReportDatabaseError closes an independent real SQL pool and proves failure is redelivered after stop.
func proveReportDatabaseError(t *testing.T, f reportFixture, b reportBrokerFixture) {
	id, _ := admittedWorkerJob(t, f)
	require.NoError(t, b.client.Publish(b.ctx, f.message(t, id)))
	_, brokenDB := acpStore(t)
	pool, err := brokenDB.DB()
	require.NoError(t, err)
	require.NoError(t, pool.Close())
	// A database failure precedes evidence and time lookup; these generated mocks expect no calls.
	worker := interviewreportusecase.Worker{Store: interviewreportrepository.Store{DB: brokenDB},
		Evidence: mocks.NewInterviewReportEvidence(t), Clock: mocks.NewClock(t), Writer: mocks.NewInterviewReportWriter(t)}
	consumer := reportbroker.Consumer{Worker: worker}
	channel, deliveries := b.consume(t)
	first := b.receive(t, deliveries)
	err = consumer.Handle(b.ctx, first)
	require.EqualError(t, err, "sql: database is closed")
	require.NoError(t, channel.Close())
	channel, deliveries = b.consume(t)
	redelivered := b.receive(t, deliveries)
	assert.True(t, redelivered.Redelivered)
	assert.Equal(t, first.MessageId, redelivered.MessageId)
	require.NoError(t, redelivered.Ack(false))
	b.assertBrokerDrained(t, channel)
	assert.Zero(t, f.count(t, `SELECT count(*) FROM interview_report_attempts WHERE job_id=?`, id))
}

// TestInterviewReportRabbitMQOutage coordinates a real Compose broker stop after connection setup.
func TestInterviewReportRabbitMQOutage(t *testing.T) {
	directory := os.Getenv("TEST_RABBITMQ_CONTROL_DIR")
	if directory == "" {
		t.Skip("isolated Compose outage coordinator required")
	}
	b := newReportBrokerFixture(t)
	f := newReportFixture(t)
	closed := b.connection.NotifyClose(make(chan *amqp.Error, 1))
	require.NoError(t, os.WriteFile(filepath.Join(directory, "ready"), nil, 0600))
	select {
	case <-closed:
	case <-b.ctx.Done():
		t.Fatal("coordinator did not stop RabbitMQ")
	}
	// Admission commits while the actual broker is offline.
	id, _ := admittedWorkerJob(t, f)
	message := f.message(t, id)
	publisher := reportbroker.Publisher{Store: f.reports, Broker: b.client, Clock: clock.System{}, BatchSize: 1, Timeout: 5 * time.Second}
	require.Error(t, publisher.Reconcile(context.Background()))
	assert.EqualValues(t, 1, f.count(t, `SELECT count(*) FROM outbox_messages WHERE id=? AND confirmed_at IS NULL AND publish_claimed_at IS NOT NULL`, message.ID))
	assert.EqualValues(t, 1, f.count(t, `SELECT count(*) FROM interview_report_jobs WHERE id=? AND state='pending' AND attempt_count=0`, id))
	assert.EqualValues(t, 1, f.count(t, `SELECT count(*) FROM interview_assignments WHERE id=? AND completed_at IS NOT NULL`, f.assignmentID))
	assert.Zero(t, f.count(t, `SELECT count(*) FROM interview_report_attempts WHERE job_id=?`, id))
}
