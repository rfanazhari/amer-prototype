package runtime

import (
	"context"
	"time"
)

const (
	ShutdownBudget = 5 * time.Second
	drainBudget    = 4 * time.Second
)

// WatchShutdown enforces the process deadline through expired. A timed-out
// cleanup remains incomplete; expired must terminate the process in main.
func WatchShutdown(ctx context.Context, started <-chan time.Time, finished <-chan struct{}, expired func()) {
	var beginning time.Time
	select {
	case <-finished:
		return
	case beginning = <-started:
	case <-ctx.Done():
		beginning = time.Now()
	}
	timer := time.NewTimer(time.Until(beginning.Add(ShutdownBudget)))
	defer timer.Stop()
	select {
	case <-finished:
		return
	case <-timer.C:
		select {
		case <-finished:
			return
		default:
		}
		expired()
	}
}
