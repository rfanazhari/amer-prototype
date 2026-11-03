package reportbroker_test

import (
	"context"
	"errors"
	"testing"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/rfanazhari/amer-prototype/application/repository/interviewreportport"
	"github.com/rfanazhari/amer-prototype/infrastructure/service/reportbroker"
	"github.com/rfanazhari/amer-prototype/test/mocks/reportbrokermocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// clientCase selects broker responses without opening a real connection.
type clientCase struct {
	name       string
	wantErr    bool
	ack        bool
	closed     bool
	lost       bool
	returned   bool
	confirmErr error
	publishErr error
}

// TestClientPublish proves persistent identity-only payloads and positive-confirm-only success.
func TestClientPublish(t *testing.T) {
	failure := errors.New("broker unavailable")
	tests := []clientCase{
		{name: "persistent identity payload and ack", ack: true},
		{name: "nack", wantErr: true},
		{name: "closed notification stream", wantErr: true, closed: true},
		{name: "lost confirm times out", wantErr: true, lost: true},
		{name: "unroutable despite ack", wantErr: true, returned: true, ack: true},
		{name: "confirm mode refused", wantErr: true, confirmErr: failure},
		{name: "publish failed", wantErr: true, publishErr: failure},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			channel := reportbrokermocks.NewChannel(t)
			confirms := make(chan amqp.Confirmation, 1)
			returns := make(chan amqp.Return, 1)
			channel.On("Confirm", false).Return(tt.confirmErr).Once()
			if tt.confirmErr == nil {
				channel.On("NotifyPublish", mock.Anything).Return(confirms).Once()
				channel.On("NotifyReturn", mock.Anything).Return(returns).Once()
				channel.On("GetNextPublishSeqNo").Return(uint64(7)).Once()
				channel.On("PublishWithContext", mock.Anything, "reports", "report-route", true, false,
					mock.MatchedBy(func(msg amqp.Publishing) bool {
						assert.JSONEq(t, `{"jobKind":"interview_report","jobId":"job","messageId":"message"}`, string(msg.Body))
						assert.Equal(t, uint8(amqp.Persistent), msg.DeliveryMode)
						assert.Equal(t, "application/json", msg.ContentType)
						assert.Equal(t, "message", msg.MessageId)
						return true
					})).Return(tt.publishErr).Once()
				if tt.closed {
					close(confirms)
				} else if !tt.lost {
					confirms <- amqp.Confirmation{DeliveryTag: 7, Ack: tt.ack}
				}
				if tt.returned {
					returns <- amqp.Return{MessageId: "message", ReplyCode: 312}
				}
			}
			client := reportbroker.Client{Channel: channel, Exchange: "reports", RoutingKey: "report-route"}
			ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
			defer cancel()
			err := client.Publish(ctx, interviewreportport.OutboxMessage{ID: "message", JobKind: "interview_report", JobID: "job"})
			assert.Equal(t, tt.wantErr, err != nil)
			if tt.lost {
				assert.ErrorIs(t, err, context.DeadlineExceeded)
			}
		})
	}
}
