package eventqueue_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/rfanazhari/amer-prototype/domain/event"
	"github.com/rfanazhari/amer-prototype/infrastructure/service/eventqueue"
	"github.com/stretchr/testify/require"
)

func TestReservationAndDrain(t *testing.T) {
	received := make(chan event.TaskCompleted, 1)
	queue := &eventqueue.Queue{Capacity: 1, Consume: func(e event.TaskCompleted) { received <- e }}
	queue.Start()
	reservation, err := queue.Reserve()
	require.NoError(t, err)
	_, err = queue.Reserve()
	require.ErrorIs(t, err, eventqueue.ErrUnavailable)
	want := event.TaskCompleted{TaskID: "task", CompletedAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)}
	reservation.Publish(want)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	require.NoError(t, queue.Shutdown(ctx))
	require.Len(t, received, 1)
	require.Equal(t, want, <-received)
	_, err = queue.Reserve()
	require.ErrorIs(t, err, eventqueue.ErrUnavailable)
}
func TestCancelAndShutdownTimeout(t *testing.T) {
	queue := &eventqueue.Queue{Capacity: 1, Consume: func(event.TaskCompleted) {}}
	queue.Start()
	reservation, err := queue.Reserve()
	require.NoError(t, err)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	require.ErrorIs(t, queue.Shutdown(ctx), context.Canceled)
	reservation.Cancel()
	ctx2, done := context.WithTimeout(context.Background(), time.Second)
	defer done()
	require.NoError(t, queue.Shutdown(ctx2))
}
func TestUnexpectedConsumerExitIsReported(t *testing.T) {
	queue := &eventqueue.Queue{Capacity: 1, Consume: func(event.TaskCompleted) { panic("subscriber failure") }}
	queue.Start()
	reservation, err := queue.Reserve()
	require.NoError(t, err)
	reservation.Publish(event.TaskCompleted{TaskID: "task"})
	select {
	case <-queue.Done():
		require.Error(t, queue.Err())
	case <-time.After(time.Second):
		t.Fatal("consumer exit was not reported")
	}
	_, err = queue.Reserve()
	require.ErrorIs(t, err, eventqueue.ErrUnavailable)
}
func TestSubscriberFailureIsLoggedWithoutRetry(t *testing.T) {
	failure := errors.New("subscriber failed")
	received := make(chan error, 2)
	calls := 0
	queue := &eventqueue.Queue{Capacity: 2, Consumer: func(context.Context, event.TaskCompleted) error { calls++; return failure }, OnError: func(err error) { received <- err }}
	queue.Start()
	for i := 0; i < 2; i++ {
		reservation, err := queue.Reserve()
		require.NoError(t, err)
		reservation.Publish(event.TaskCompleted{TaskID: "task"})
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	require.NoError(t, queue.Shutdown(ctx))
	require.Equal(t, 2, calls)
	require.Len(t, received, 2)
	require.ErrorIs(t, <-received, failure)
}
func TestForcedStopCancelsConsumerAndReportsFailure(t *testing.T) {
	entered := make(chan struct{})
	queue := &eventqueue.Queue{Capacity: 1, Consumer: func(ctx context.Context, _ event.TaskCompleted) error { close(entered); <-ctx.Done(); return ctx.Err() }}
	queue.Start()
	reservation, err := queue.Reserve()
	require.NoError(t, err)
	reservation.Publish(event.TaskCompleted{TaskID: "task"})
	select {
	case <-entered:
	case <-time.After(time.Second):
		t.Fatal("consumer did not start")
	}
	queue.ForceStop()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	require.ErrorIs(t, queue.Shutdown(ctx), context.Canceled)
}
