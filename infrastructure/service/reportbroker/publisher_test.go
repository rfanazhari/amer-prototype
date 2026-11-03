package reportbroker_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/rfanazhari/amer-prototype/application/repository/interviewreportport"
	"github.com/rfanazhari/amer-prototype/infrastructure/service/reportbroker"
	"github.com/rfanazhari/amer-prototype/test/mocks"
	"github.com/rfanazhari/amer-prototype/test/mocks/reportbrokermocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// publishCase specifies one durable publication outcome.
type publishCase struct {
	name       string
	wantErr    bool
	claimErr   error
	publishErr error
	confirmErr error
	empty      bool
}

// TestPublisherReconcile covers confirmation ordering and failed publication isolation.
func TestPublisherReconcile(t *testing.T) {
	failure := errors.New("dependency unavailable")
	now := time.Date(2026, 11, 3, 0, 0, 0, 123000000, time.UTC)
	message := interviewreportport.OutboxMessage{ID: "message", JobKind: "interview_report", JobID: "job", CreatedAt: now}
	tests := []publishCase{
		{name: "confirmed after publish"},
		{name: "claim error never publishes", wantErr: true, claimErr: failure},
		{name: "lost confirm leaves row unconfirmed", wantErr: true, publishErr: failure},
		{name: "database failure leaves intent retryable", wantErr: true, confirmErr: failure},
		{name: "empty batch never publishes", empty: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := reportbrokermocks.NewOutbox(t)
			broker := reportbrokermocks.NewBroker(t)
			clock := mocks.NewClock(t)
			clock.On("Now").Return(now)
			rows := []interviewreportport.OutboxMessage{message}
			if tt.empty {
				rows = nil
			}
			store.On("ClaimOutbox", mock.Anything, 10, now).Return(rows, tt.claimErr).Once()
			if tt.claimErr == nil && !tt.empty {
				publish := broker.On("Publish", mock.MatchedBy(func(ctx context.Context) bool {
					_, ok := ctx.Deadline()
					return ok
				}), message).Return(tt.publishErr).Once()
				if tt.publishErr == nil {
					store.On("ConfirmOutbox", mock.Anything, message, now).Return(tt.confirmErr).Once().NotBefore(publish)
				}
			}
			publisher := reportbroker.Publisher{Store: store, Broker: broker, Clock: clock, BatchSize: 10, Timeout: time.Second}
			err := publisher.Reconcile(context.Background())
			assert.Equal(t, tt.wantErr, err != nil)
			if tt.wantErr {
				assert.ErrorIs(t, err, failure)
			}
		})
	}
}

// TestPublisherReconcileReplay proves a lost confirm reuses the durable message identity.
func TestPublisherReconcileReplay(t *testing.T) {
	// replayCase selects the number of identical reconciliation passes.
	type replayCase struct {
		name    string
		wantErr bool
	}
	for _, tt := range []replayCase{{name: "retries same identity after uncertain delivery"}} {
		t.Run(tt.name, func(t *testing.T) {
			now := time.Date(2026, 11, 3, 0, 0, 0, 0, time.UTC)
			message := interviewreportport.OutboxMessage{ID: "same-message", JobKind: "interview_report", JobID: "same-job"}
			store := reportbrokermocks.NewOutbox(t)
			broker := reportbrokermocks.NewBroker(t)
			clock := mocks.NewClock(t)
			clock.On("Now").Return(now)
			store.On("ClaimOutbox", mock.Anything, 1, now).Return([]interviewreportport.OutboxMessage{message}, nil).Twice()
			first := broker.On("Publish", mock.Anything, message).Return(context.DeadlineExceeded).Once()
			second := broker.On("Publish", mock.Anything, message).Return(nil).Once().NotBefore(first)
			store.On("ConfirmOutbox", mock.Anything, message, now).Return(nil).Once().NotBefore(second)
			publisher := reportbroker.Publisher{Store: store, Broker: broker, Clock: clock, BatchSize: 1, Timeout: time.Second}
			assert.ErrorIs(t, publisher.Reconcile(context.Background()), context.DeadlineExceeded)
			assert.Equal(t, tt.wantErr, publisher.Reconcile(context.Background()) != nil)
		})
	}
}
