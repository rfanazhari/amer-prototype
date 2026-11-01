package runtime

import (
	"context"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/SALT-Indonesia/salt-pkg/logmanager"
	"github.com/rfanazhari/amer-prototype/delivery/http/handler/taskhandler"
)

// httpRuntime owns HTTP admission, active handlers and the independent request cancellation lifetime.
type httpRuntime struct {
	server   *http.Server
	cancel   context.CancelFunc
	mu       sync.Mutex
	stopping bool
	active   sync.WaitGroup
	done     chan struct{}
}

// HTTPServer assembles the HTTP server with logging context, admission gating and a request lifetime that permits graceful shutdown.
func HTTPServer(handler http.Handler, logs *logmanager.Application) *httpRuntime {
	workCtx, cancel := context.WithCancel(context.Background())
	transport := &httpRuntime{cancel: cancel, done: make(chan struct{})}
	transport.server = &http.Server{BaseContext: func(net.Listener) context.Context { return workCtx }, Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		transport.mu.Lock()
		if transport.stopping {
			transport.mu.Unlock()
			taskhandler.ShuttingDown(w, r)
			return
		}
		transport.active.Add(1)
		transport.mu.Unlock()
		defer transport.active.Done()
		txn := logs.StartHttp("", r.Method+" "+r.URL.Path)
		defer txn.End()
		w = txn.SetWebResponseHttp(w)
		handler.ServeHTTP(w, r.WithContext(txn.ToContext(r.Context())))
	}), ReadHeaderTimeout: 5 * time.Second}
	return transport
}

// StopAdmission atomically rejects new work and starts observing completion of already admitted handlers.
func (h *httpRuntime) StopAdmission() {
	h.mu.Lock()
	if !h.stopping {
		h.stopping = true
		go func() { h.active.Wait(); close(h.done) }()
	}
	h.mu.Unlock()
}

// ServeHTTP exposes the admission-gated handler for verification of new-work rejection.
func (h *httpRuntime) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.server.Handler.ServeHTTP(w, r)
}

// wait observes active-handler completion after admission is stopped, without treating a deadline as completed work.
func (h *httpRuntime) wait(ctx context.Context) error {
	select {
	case <-h.done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
