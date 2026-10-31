package smoke_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/SALT-Indonesia/salt-pkg/logmanager"
	"github.com/google/uuid"
	"github.com/rfanazhari/amer-prototype/domain/event"
	appruntime "github.com/rfanazhari/amer-prototype/infrastructure/runtime"
	"github.com/stretchr/testify/require"
)

type taskResponse struct {
	ID          string  `json:"id"`
	Status      string  `json:"status"`
	DueDate     *string `json:"dueDate"`
	CompletedAt *string `json:"completedAt"`
}
type errorResponse struct {
	Code string `json:"code"`
}
type response struct {
	Status string         `json:"status"`
	Data   taskResponse   `json:"data"`
	Error  *errorResponse `json:"error"`
}
type fixture struct {
	url      string
	shutdown <-chan time.Time
	stop     func() error
}

func start(t *testing.T, subscriber func(context.Context, event.TaskCompleted) error) fixture {
	t.Helper()
	return startWithListener(t, subscriber, nil)
}
func startWithListener(t *testing.T, subscriber func(context.Context, event.TaskCompleted) error, listen func(context.Context, string, string) (net.Listener, error)) fixture {
	t.Helper()
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL must identify an authorized local PostgreSQL database")
	}
	ctx, cancel := context.WithCancel(context.Background())
	ready := make(chan net.Addr, 1)
	result := make(chan error, 1)
	shutdown := make(chan time.Time, 1)
	logs := logmanager.NewApplication(logmanager.WithAppName("smoke-test"), logmanager.WithSkipHeaders())
	go func() {
		result <- appruntime.Run(ctx, appruntime.Config{DatabaseURL: databaseURL, Address: "127.0.0.1:0", Ready: ready, ShutdownStarted: shutdown, TaskCompleted: subscriber, Listen: listen}, logs)
	}()
	var once sync.Once
	var stopErr error
	stop := func() error {
		once.Do(func() {
			cancel()
			select {
			case stopErr = <-result:
			case <-time.After(6 * time.Second):
				stopErr = errors.New("runtime did not terminate within shutdown budget")
			}
		})
		return stopErr
	}
	t.Cleanup(func() { _ = stop() })
	select {
	case addr := <-ready:
		return fixture{url: "http://" + addr.String(), shutdown: shutdown, stop: stop}
	case err := <-result:
		cancel()
		t.Fatalf("startup: %v", err)
	case <-time.After(15 * time.Second):
		cancel()
		t.Fatal("startup timeout")
	}
	return fixture{}
}
func request(t *testing.T, url, method, path, body string, status int) response {
	t.Helper()
	client := http.Client{Timeout: 2 * time.Second}
	req, err := http.NewRequest(method, url+path, bytes.NewBufferString(body))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer func() { require.NoError(t, resp.Body.Close()) }()
	data, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.Equal(t, status, resp.StatusCode, string(data))
	require.Contains(t, resp.Header.Get("Content-Type"), "application/json")
	var result response
	require.NoError(t, json.Unmarshal(data, &result))
	return result
}
func complete(t *testing.T, service fixture) string {
	t.Helper()
	created := request(t, service.url, "POST", "/tasks", `{"title":"integration task"}`, 201)
	id := created.Data.ID
	request(t, service.url, "POST", "/tasks/"+id+"/start", "{}", 200)
	request(t, service.url, "POST", "/tasks/"+id+"/complete", "", 200)
	return id
}

func TestHealthAndShutdown(t *testing.T) {
	service := start(t, nil)
	result := request(t, service.url, "GET", "/health", "", 200)
	require.Equal(t, "success", result.Status)
	require.Nil(t, result.Error)
	require.NoError(t, service.stop())
}
func TestTaskAPIFlow(t *testing.T) {
	events := make(chan event.TaskCompleted, 2)
	service := start(t, func(_ context.Context, e event.TaskCompleted) error { events <- e; return nil })
	created := request(t, service.url, "POST", "/tasks", `{"title":" Prepare report "}`, 201)
	id := created.Data.ID
	require.NoError(t, uuid.Validate(id))
	require.Equal(t, "todo", created.Data.Status)
	require.Nil(t, created.Data.DueDate)
	require.Nil(t, created.Data.CompletedAt)
	request(t, service.url, "POST", "/tasks/"+id+"/complete", "{}", 409)
	request(t, service.url, "POST", "/tasks/"+id+"/start", "", 200)
	due := time.Now().UTC().Add(time.Hour).Truncate(time.Millisecond).Format("2006-01-02T15:04:05.000Z")
	updated := request(t, service.url, "PATCH", "/tasks/"+id+"/due-date", `{"dueDate":"`+due+`"}`, 200)
	require.Equal(t, &due, updated.Data.DueDate)
	cleared := request(t, service.url, "PATCH", "/tasks/"+id+"/due-date", `{"dueDate":null}`, 200)
	require.Nil(t, cleared.Data.DueDate)
	done := request(t, service.url, "POST", "/tasks/"+id+"/complete", "{}", 200)
	require.Equal(t, "done", done.Data.Status)
	require.NotNil(t, done.Data.CompletedAt)
	detail := request(t, service.url, "GET", "/tasks/"+id, "", 200)
	require.Equal(t, done.Data, detail.Data)
	request(t, service.url, "POST", "/tasks/"+id+"/complete", "{}", 409)
	request(t, service.url, "POST", "/tasks/"+id+"/start", "{}", 409)
	request(t, service.url, "PATCH", "/tasks/"+id+"/due-date", `{"dueDate":null}`, 409)
	request(t, service.url, "GET", "/tasks/"+uuid.NewString(), "", 404)
	request(t, service.url, "GET", "/tasks/not-a-uuid", "", 400)
	require.NoError(t, service.stop())
	require.Len(t, events, 1)
	require.Equal(t, id, (<-events).TaskID)
}
func TestShutdownDrainsAcceptedEvent(t *testing.T) {
	entered := make(chan struct{})
	release := make(chan struct{})
	consumed := make(chan struct{})
	service := start(t, func(ctx context.Context, _ event.TaskCompleted) error {
		close(entered)
		select {
		case <-release:
			close(consumed)
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	})
	complete(t, service)
	select {
	case <-entered:
	case <-time.After(time.Second):
		t.Fatal("subscriber not started")
	}
	stopped := make(chan error, 1)
	go func() { stopped <- service.stop() }()
	select {
	case <-service.shutdown:
	case <-time.After(time.Second):
		t.Fatal("shutdown not started")
	}
	close(release)
	require.NoError(t, <-stopped)
	select {
	case <-consumed:
	default:
		t.Fatal("accepted event not consumed")
	}
}
func TestShutdownReportsBlockedSubscriber(t *testing.T) {
	entered := make(chan struct{})
	service := start(t, func(ctx context.Context, _ event.TaskCompleted) error { close(entered); <-ctx.Done(); return ctx.Err() })
	complete(t, service)
	select {
	case <-entered:
	case <-time.After(time.Second):
		t.Fatal("subscriber not started")
	}
	beginning := time.Now()
	err := service.stop()
	require.ErrorIs(t, err, context.DeadlineExceeded)
	require.Less(t, time.Since(beginning), appruntime.ShutdownBudget)
}

func TestPersistenceAcrossRuntimeRestart(t *testing.T) {
	first := start(t, nil)
	created := request(t, first.url, "POST", "/tasks", `{"title":"persistent task"}`, 201)
	require.NoError(t, first.stop())
	second := start(t, nil)
	loaded := request(t, second.url, "GET", "/tasks/"+created.Data.ID, "", 200)
	require.Equal(t, created.Data, loaded.Data)
	require.NoError(t, second.stop())
}

func TestUnexpectedConsumerExitStopsComposition(t *testing.T) {
	service := start(t, func(context.Context, event.TaskCompleted) error { panic("verification worker failure") })
	id := complete(t, service)
	select {
	case <-service.shutdown:
	case <-time.After(time.Second):
		t.Fatal("worker failure was not supervised")
	}
	require.ErrorContains(t, service.stop(), "event consumer stopped")
	next := start(t, nil)
	persisted := request(t, next.url, "GET", "/tasks/"+id, "", 200)
	require.Equal(t, "done", persisted.Data.Status)
	require.NoError(t, next.stop())
}
func TestUnexpectedHTTPExitStopsComposition(t *testing.T) {
	bound := make(chan net.Listener, 1)
	service := startWithListener(t, nil, func(ctx context.Context, network, address string) (net.Listener, error) {
		listener, err := (&net.ListenConfig{}).Listen(ctx, network, address)
		if err == nil {
			bound <- listener
		}
		return listener, err
	})
	listener := <-bound
	require.NoError(t, listener.Close())
	select {
	case <-service.shutdown:
	case <-time.After(time.Second):
		t.Fatal("HTTP failure was not supervised")
	}
	require.ErrorContains(t, service.stop(), "HTTP server")
}
