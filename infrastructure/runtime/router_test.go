package runtime_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/SALT-Indonesia/salt-pkg/logmanager"
	appruntime "github.com/rfanazhari/amer-prototype/infrastructure/runtime"
	"github.com/rfanazhari/amer-prototype/pkg/errs"
	"github.com/stretchr/testify/require"
)

// routeCase is one composed-router request and the handler that must receive it.
type routeCase struct {
	name       string
	method     string
	path       string
	body       string
	wantStatus int
	wantTask   bool
	wantCode   string
}

// admissionCase is one request that shutdown admission must reject before a route handler.
type admissionCase struct {
	name   string
	method string
	path   string
	body   string
}

// recordedTaskHandler records whether Task received a request and the path it saw.
type recordedTaskHandler struct {
	called atomic.Bool
	path   atomic.Value
	method atomic.Value
}

// ServeHTTP records the request and returns Task-like status codes for health, tasks, and unknown paths.
func (h *recordedTaskHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.called.Store(true)
	h.path.Store(r.URL.Path)
	h.method.Store(r.Method)
	switch {
	case r.URL.Path == "/health":
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"success","data":{}}`))
	case strings.HasPrefix(r.URL.Path, "/tasks"):
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"status":"success","data":{"id":"task"}}`))
	default:
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"status":"error","data":{},"error":{"code":"NOT_FOUND","message":"not found"}}`))
	}
}

// envelopeView is the decoded public JSON envelope used by composition assertions.
type envelopeView struct {
	Status string          `json:"status"`
	Data   json.RawMessage `json:"data"`
	Error  *struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

// composeRouter wires only the Task double, because the process serves no prototype voice handler.
func composeRouter(t *testing.T, task http.Handler) appruntime.Router {
	t.Helper()
	return appruntime.Router{Task: task}
}

// TestRemovedVoicePathsFallThroughToTask checks the prototype voice paths are unrouted and reach the Task not-found response.
func TestRemovedVoicePathsFallThroughToTask(t *testing.T) {
	tests := []routeCase{
		{name: "voice page", method: http.MethodGet, path: "/voice-session", wantStatus: http.StatusNotFound, wantTask: true, wantCode: errs.NotFound},
		{name: "voice setup", method: http.MethodPost, path: "/voice-sessions", body: `{"sdp":"offer-sdp"}`, wantStatus: http.StatusNotFound, wantTask: true, wantCode: errs.NotFound},
		{name: "voice setup wrong method", method: http.MethodGet, path: "/voice-sessions", wantStatus: http.StatusNotFound, wantTask: true, wantCode: errs.NotFound},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			task := &recordedTaskHandler{}
			router := composeRouter(t, task)
			recorder := httptest.NewRecorder()
			router.ServeHTTP(recorder, httptest.NewRequest(tt.method, tt.path, strings.NewReader(tt.body)))
			require.Equal(t, tt.wantStatus, recorder.Code)
			require.True(t, task.called.Load())
			require.Equal(t, tt.path, task.path.Load())
			var view envelopeView
			require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &view))
			require.Equal(t, "error", view.Status)
			require.Equal(t, tt.wantCode, view.Error.Code)
		})
	}
}

// TestTaskRouteDispatch checks health, task and unknown paths stay on Task.
func TestTaskRouteDispatch(t *testing.T) {
	tests := []routeCase{
		{name: "health", method: http.MethodGet, path: "/health", wantStatus: http.StatusOK, wantTask: true},
		{name: "tasks", method: http.MethodPost, path: "/tasks", body: `{"title":"demo"}`, wantStatus: http.StatusCreated, wantTask: true},
		{name: "unknown", method: http.MethodGet, path: "/missing", wantStatus: http.StatusNotFound, wantTask: true, wantCode: errs.NotFound},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			task := &recordedTaskHandler{}
			router := composeRouter(t, task)
			recorder := httptest.NewRecorder()
			request := httptest.NewRequest(tt.method, tt.path, strings.NewReader(tt.body))
			router.ServeHTTP(recorder, request)
			require.Equal(t, tt.wantStatus, recorder.Code)
			require.Equal(t, tt.wantTask, task.called.Load())
			if tt.wantTask {
				require.Equal(t, tt.path, task.path.Load())
				require.Equal(t, tt.method, task.method.Load())
			}
			if tt.wantCode != "" {
				var view envelopeView
				require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &view))
				require.Equal(t, "error", view.Status)
				require.Equal(t, tt.wantCode, view.Error.Code)
			}
		})
	}
}

// TestShutdownAdmissionRejectsNewWork checks that stopped admission returns SHUTTING_DOWN before route handlers.
func TestShutdownAdmissionRejectsNewWork(t *testing.T) {
	tests := []admissionCase{
		{name: "removed voice setup path", method: http.MethodPost, path: "/voice-sessions", body: `{"sdp":"offer-sdp"}`},
		{name: "removed voice page path", method: http.MethodGet, path: "/voice-session"},
		{name: "tasks", method: http.MethodPost, path: "/tasks", body: `{"title":"demo"}`},
		{name: "health", method: http.MethodGet, path: "/health"},
		{name: "unknown", method: http.MethodGet, path: "/missing"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			task := &recordedTaskHandler{}
			router := composeRouter(t, task)
			logs := logmanager.NewApplication(logmanager.WithAppName("runtime-test"))
			transport := appruntime.HTTPServer(router, logs)
			transport.StopAdmission()
			recorder := httptest.NewRecorder()
			request := httptest.NewRequest(tt.method, tt.path, strings.NewReader(tt.body))
			transport.ServeHTTP(recorder, request)
			require.Equal(t, http.StatusServiceUnavailable, recorder.Code)
			require.False(t, task.called.Load())
			var view envelopeView
			require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &view))
			require.Equal(t, "error", view.Status)
			require.Equal(t, errs.ShuttingDown, view.Error.Code)
		})
	}
}
