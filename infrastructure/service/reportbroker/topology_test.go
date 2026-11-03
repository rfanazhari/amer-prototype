package reportbroker_test

import (
	"errors"
	"testing"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/rfanazhari/amer-prototype/infrastructure/service/reportbroker"
	"github.com/rfanazhari/amer-prototype/test/mocks/reportbrokermocks"
	"github.com/stretchr/testify/assert"
)

// topologyCase selects the failing broker declaration stage or successful provisioning.
type topologyCase struct {
	name    string
	wantErr bool
	failAt  int
}

// TestTopologyDeclare verifies fixed isolation names, durable quorum queues and fail-closed ordering.
func TestTopologyDeclare(t *testing.T) {
	tests := []topologyCase{
		{name: "approved topology"},
		{name: "exchange failure", wantErr: true, failAt: 1},
		{name: "quarantine failure", wantErr: true, failAt: 2},
		{name: "binding failure", wantErr: true, failAt: 3},
		{name: "source failure", wantErr: true, failAt: 4},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			channel := reportbrokermocks.NewTopologyChannel(t)
			failure := errors.New("topology unavailable")
			stepErr := func(step int) error {
				if tt.failAt == step {
					return failure
				}
				return nil
			}
			channel.On("ExchangeDeclare", "interview-report.dlx", "direct", true, false, false, false, amqp.Table(nil)).Return(stepErr(1)).Once()
			if tt.failAt == 0 || tt.failAt > 1 {
				channel.On("QueueDeclare", "interview-report.invalid", true, false, false, false,
					amqp.Table{"x-queue-type": "quorum"}).Return(amqp.Queue{}, stepErr(2)).Once()
			}
			if tt.failAt == 0 || tt.failAt > 2 {
				channel.On("QueueBind", "interview-report.invalid", "malformed", "interview-report.dlx", false, amqp.Table(nil)).Return(stepErr(3)).Once()
			}
			if tt.failAt == 0 || tt.failAt > 3 {
				channel.On("QueueDeclare", "reports", true, false, false, false, amqp.Table{
					"x-queue-type": "quorum", "x-dead-letter-exchange": "interview-report.dlx",
					"x-dead-letter-routing-key": "malformed", "x-dead-letter-strategy": "at-least-once",
					"x-overflow": "reject-publish", "x-delivery-limit": -1,
				}).Return(amqp.Queue{}, stepErr(4)).Once()
			}
			err := (reportbroker.Topology{Channel: channel}).Declare("reports")
			assert.Equal(t, tt.wantErr, err != nil)
			if tt.wantErr {
				assert.ErrorIs(t, err, failure)
			}
		})
	}
}
