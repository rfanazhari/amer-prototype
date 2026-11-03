package reportbroker_test

import (
	"context"
	"errors"
	"testing"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/rfanazhari/amer-prototype/infrastructure/service/reportbroker"
	"github.com/rfanazhari/amer-prototype/test/mocks/reportbrokermocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// consumerCase selects transport validation, rejection and worker failure behavior.
type consumerCase struct {
	name      string
	wantErr   bool
	body      string
	malformed bool
	rejectErr error
	workerErr error
	canceled  bool
}

// TestConsumerHandle isolates malformed deliveries without ACK, requeue, or worker side effects.
func TestConsumerHandle(t *testing.T) {
	failure := errors.New("channel or database unavailable")
	valid := `{"jobKind":"interview_report","jobId":"job","messageId":"message"}`
	tests := []consumerCase{
		{name: "invalid JSON", body: `{`, malformed: true},
		{name: "null body", body: `null`, malformed: true},
		{name: "wrong field type", body: `{"jobId":1}`, malformed: true},
		{name: "wrong job kind", body: `{"jobKind":"document","jobId":"job","messageId":"message"}`, malformed: true},
		{name: "missing job", body: `{"jobKind":"interview_report","messageId":"message"}`, malformed: true},
		{name: "missing message", body: `{"jobKind":"interview_report","jobId":"job"}`, malformed: true},
		{name: "message identity mismatch", body: `{"jobKind":"interview_report","jobId":"job","messageId":"other"}`, malformed: true},
		{name: "reject failure is propagated", wantErr: true, body: `{`, malformed: true, rejectErr: failure},
		{name: "valid job acknowledges", body: valid},
		{name: "database failure stays unacknowledged", wantErr: true, body: valid, workerErr: failure},
		{name: "cancellation stays unacknowledged", wantErr: true, body: valid, canceled: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			worker := reportbrokermocks.NewJobExecutor(t)
			ack := reportbrokermocks.NewAcknowledger(t)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			if tt.canceled {
				cancel()
			} else if tt.malformed {
				ack.On("Reject", uint64(7), false).Return(tt.rejectErr).Once()
			} else {
				worker.On("Execute", mock.Anything, "job").Return(tt.workerErr).Once()
				if tt.workerErr == nil {
					ack.On("Ack", uint64(7), false).Return(nil).Once()
				}
			}
			consumer := reportbroker.Consumer{Worker: worker}
			err := consumer.Handle(ctx, amqp.Delivery{Acknowledger: ack, DeliveryTag: 7, MessageId: "message", Body: []byte(tt.body)})
			assert.Equal(t, tt.wantErr, err != nil)
			if tt.rejectErr != nil {
				assert.ErrorIs(t, err, tt.rejectErr)
			}
			if tt.workerErr != nil {
				assert.ErrorIs(t, err, tt.workerErr)
			}
			if tt.canceled {
				assert.ErrorIs(t, err, context.Canceled)
			}
		})
	}
}
