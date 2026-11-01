package runtime

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/SALT-Indonesia/salt-pkg/logmanager"
	"github.com/rfanazhari/amer-prototype/delivery/http/candidatelanding"
	"github.com/rfanazhari/amer-prototype/domain/event"
)

// Config selects required resources and listener settings, with narrow readiness, shutdown and lifecycle seams for verification.
type Config struct {
	DatabaseURL string
	Address     string
	Ready       chan<- net.Addr
	// Buffered notification for the entry point's process watchdog.
	ShutdownStarted chan<- time.Time
	TaskCompleted   func(context.Context, event.TaskCompleted) error
	// Listen is an optional lifecycle seam; nil uses net.ListenConfig.
	Listen func(context.Context, string, string) (net.Listener, error)
	// OpenAIAPIKey is optional; an empty value must not block process startup.
	OpenAIAPIKey string
	// OpenAIRealtimeModel selects the Realtime model; empty uses the adapter default at composition.
	OpenAIRealtimeModel string
	// OpenAIRealtimeVoice selects the Realtime voice; empty uses the adapter default at composition.
	OpenAIRealtimeVoice string
	// OpenAIRealtimeURL selects the Realtime calls URL; empty uses the adapter default at composition.
	OpenAIRealtimeURL string
	// AccessBootstrapAdminName is the first ADMIN display name used only when no accounts exist.
	AccessBootstrapAdminName string
	// AccessBootstrapAdminEmail is the canonical first ADMIN email used only when no accounts exist.
	AccessBootstrapAdminEmail string
	// AccessBootstrapAdminPassword is the first ADMIN secret; it is never logged.
	AccessBootstrapAdminPassword string
	// AccessHTTPOrigin is the exact allowed Origin for administrative JSON requests.
	AccessHTTPOrigin string
	// AccessCSRFSecret is the HMAC key for administrative CSRF values; it is never logged.
	AccessCSRFSecret string
}

// Run acquires required resources, assembles the Task feature and supervises HTTP and event processing. It joins runtime and cleanup errors; the executable must install WatchShutdown to enforce process termination.
func Run(ctx context.Context, cfg Config, logs *logmanager.Application) (result error) {
	if cfg.DatabaseURL == "" || cfg.Address == "" || logs == nil {
		return errors.New("database URL, address and logging are required")
	}
	startup := logs.Start("", "service startup", logmanager.TxnTypeOther)
	startupComplete := false
	defer func() {
		if !startupComplete {
			// Keep configuration and underlying driver diagnostics out of lifecycle records.
			startup.NoticeError(errors.New("service startup failed"))
		}
	}()
	startupLogCtx := startup.ToContext(ctx)
	logmanager.InfoWithContext(startupLogCtx, "service starting", map[string]string{"service": logs.Service()})
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := ValidateAccessSettings(cfg); err != nil {
		return err
	}
	var shutdownAt time.Time
	beginShutdown := func() {
		if !shutdownAt.IsZero() {
			return
		}
		shutdownAt = time.Now()
		if cfg.ShutdownStarted != nil {
			select {
			case cfg.ShutdownStarted <- shutdownAt:
			default:
			}
		}
	}
	startupCtx, cancelStartup := context.WithTimeout(ctx, 10*time.Second)
	defer cancelStartup()
	db, resources, err := openDatabase(startupCtx, cfg.DatabaseURL, beginShutdown)
	if err != nil {
		return err
	}
	logmanager.InfoWithContext(startupLogCtx, "database connected and schema compatible", map[string]string{"service": logs.Service(), "schema_version": "2"})
	defer func() {
		beginShutdown()
		cleanupCtx, cancel := context.WithDeadline(context.Background(), shutdownAt.Add(ShutdownBudget))
		defer cancel()
		if err := resources.close(cleanupCtx); err != nil {
			result = errors.Join(result, err)
		}
		if time.Now().After(shutdownAt.Add(ShutdownBudget)) {
			result = errors.Join(result, fmt.Errorf("database cleanup exceeded shutdown budget: %w", context.DeadlineExceeded))
		}
	}()
	listen := cfg.Listen
	if listen == nil {
		listen = (&net.ListenConfig{}).Listen
	}
	listener, err := listen(startupCtx, "tcp", cfg.Address)
	if err != nil {
		return fmt.Errorf("listen: %w", err)
	}
	if err = startupCtx.Err(); err != nil {
		return errors.Join(err, listener.Close())
	}
	if err = bootstrapAccess(startupCtx, db, cfg); err != nil {
		return fmt.Errorf("access bootstrap: %w", err)
	}
	logmanager.InfoWithContext(startupLogCtx, "administrative bootstrap complete", map[string]string{"service": logs.Service()})
	taskHandler, queue := assembleTask(db, logs, cfg.TaskCompleted)
	page, setup := assembleVoice(cfg, logs)
	handler := Router{Page: page, Setup: setup, Task: taskHandler, Access: assembleAccess(db, cfg, logs), Landing: candidatelanding.Handler{}}
	transport := HTTPServer(handler, logs)
	defer transport.cancel()
	queue.Start()
	serveErr := make(chan error, 1)
	go func() { serveErr <- transport.server.Serve(listener) }()
	logmanager.InfoWithContext(startupLogCtx, "http server ready", map[string]string{
		"service": logs.Service(), "address": listener.Addr().String(), "schema_version": "2", "event_queue": "in_process",
	})
	startup.End()
	startupComplete = true
	if cfg.Ready != nil {
		select {
		case cfg.Ready <- listener.Addr():
		default:
		}
	}
	var cause error
	serveExited := false
	select {
	case <-ctx.Done():
	case err = <-serveErr:
		serveExited = true
		if err == nil || errors.Is(err, http.ErrServerClosed) {
			cause = errors.New("HTTP server exited unexpectedly")
		} else {
			cause = fmt.Errorf("HTTP server: %w", err)
		}
	case <-queue.Done():
		cause = queue.Err()
		if cause == nil {
			cause = errors.New("event consumer exited unexpectedly")
		}
	}
	beginShutdown()
	shutdownCtx, cancelShutdown := context.WithDeadline(context.Background(), shutdownAt.Add(ShutdownBudget))
	defer cancelShutdown()
	drainCtx, cancelDrain := context.WithDeadline(shutdownCtx, shutdownAt.Add(drainBudget))
	defer cancelDrain()
	transport.StopAdmission()
	if err = transport.server.Shutdown(drainCtx); err != nil {
		cause = errors.Join(cause, fmt.Errorf("HTTP drain: %w", err))
		transport.cancel()
		if closeErr := transport.server.Close(); closeErr != nil {
			cause = errors.Join(cause, fmt.Errorf("HTTP forced close: %w", closeErr))
		}
	}
	if err = transport.wait(shutdownCtx); err != nil {
		cause = errors.Join(cause, fmt.Errorf("active requests did not terminate: %w", err))
	}
	if !serveExited {
		select {
		case err = <-serveErr:
			if err != nil && !errors.Is(err, http.ErrServerClosed) {
				cause = errors.Join(cause, fmt.Errorf("HTTP server: %w", err))
			}
		case <-shutdownCtx.Done():
			cause = errors.Join(cause, fmt.Errorf("HTTP server termination: %w", shutdownCtx.Err()))
		}
	}
	if err = queue.Shutdown(drainCtx); err != nil {
		cause = errors.Join(cause, fmt.Errorf("event drain incomplete: %w", err))
		queue.ForceStop()
		if waitErr := queue.Shutdown(shutdownCtx); waitErr != nil {
			cause = errors.Join(cause, fmt.Errorf("event consumer did not terminate: %w", waitErr))
		}
	}
	return cause
}
