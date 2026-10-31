package eventqueue

import (
	"context"
	"fmt"
	"sync"

	"github.com/rfanazhari/amer-prototype/application/service"
	"github.com/rfanazhari/amer-prototype/domain/event"
)

var ErrUnavailable = service.ErrEventUnavailable

type Queue struct {
	Capacity  int
	Consume   func(event.TaskCompleted)
	Consumer  func(context.Context, event.TaskCompleted) error
	OnError   func(error)
	workerCtx context.Context
	cancel    context.CancelFunc
	mu        sync.Mutex
	started   bool
	closing   bool
	slots     chan struct{}
	events    chan event.TaskCompleted
	pending   sync.WaitGroup
	done      chan struct{}
	err       error
}

type Reservation struct {
	queue *Queue
	once  sync.Once
}

func (q *Queue) Start() {
	q.mu.Lock()
	defer q.mu.Unlock()
	if q.started {
		return
	}
	q.started = true
	q.workerCtx, q.cancel = context.WithCancel(context.Background())
	q.slots = make(chan struct{}, q.Capacity)
	q.events = make(chan event.TaskCompleted, q.Capacity)
	q.done = make(chan struct{})
	go func() {
		defer func() {
			q.cancel()
			q.mu.Lock()
			if recovered := recover(); recovered != nil {
				q.err = fmt.Errorf("event consumer stopped: %v", recovered)
				q.closing = true
			}
			close(q.done)
			q.mu.Unlock()
		}()
		for {
			select {
			case <-q.workerCtx.Done():
				return
			case e, open := <-q.events:
				if !open {
					return
				}
				<-q.slots
				if q.Consumer != nil {
					if err := q.Consumer(q.workerCtx, e); err != nil && q.OnError != nil {
						q.OnError(err)
					}
				} else {
					q.Consume(e)
				}
			}
		}
	}()
}

// ForceStop cancels the consumer after a drain failure. Accepted events may be lost.
func (q *Queue) ForceStop() {
	q.mu.Lock()
	defer q.mu.Unlock()
	if q.started {
		q.closing = true
		if q.err == nil {
			q.err = context.Canceled
		}
		q.cancel()
	}
}

func (q *Queue) Done() <-chan struct{} {
	q.mu.Lock()
	defer q.mu.Unlock()
	return q.done
}

func (q *Queue) Err() error {
	q.mu.Lock()
	defer q.mu.Unlock()
	return q.err
}

func (q *Queue) Reserve() (service.Reservation, error) {
	q.mu.Lock()
	defer q.mu.Unlock()
	if !q.started || q.closing {
		return nil, ErrUnavailable
	}
	select {
	case q.slots <- struct{}{}:
		q.pending.Add(1)
		return &Reservation{queue: q}, nil
	default:
		return nil, ErrUnavailable
	}
}

func (r *Reservation) Publish(e event.TaskCompleted) {
	r.once.Do(func() { r.queue.events <- e; r.queue.pending.Done() })
}

func (r *Reservation) Cancel() {
	r.once.Do(func() { <-r.queue.slots; r.queue.pending.Done() })
}

func (q *Queue) Shutdown(ctx context.Context) error {
	q.mu.Lock()
	if !q.started {
		q.mu.Unlock()
		return nil
	}
	if !q.closing {
		q.closing = true
		go func() { q.pending.Wait(); close(q.events) }()
	}
	done := q.done
	q.mu.Unlock()
	select {
	case <-done:
		return q.Err()
	case <-ctx.Done():
		return ctx.Err()
	}
}
