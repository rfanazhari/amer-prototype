package runtime_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/rfanazhari/amer-prototype/delivery/http/candidatelanding"
	"github.com/rfanazhari/amer-prototype/delivery/http/handler/accesshandler"
	appruntime "github.com/rfanazhari/amer-prototype/infrastructure/runtime"
	"github.com/stretchr/testify/require"
)

// recordedConfigHandler records whether configuration routes reached the configuration handler.
type recordedConfigHandler struct{ called atomic.Bool }

// ServeHTTP records the request and returns a configuration-like success envelope.
func (h *recordedConfigHandler) ServeHTTP(w http.ResponseWriter, _ *http.Request) {
	h.called.Store(true)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"status":"success","data":{}}`))
}

// candidateRouteCase names one composed-router candidate path outcome.
type candidateRouteCase struct {
	name       string
	method     string
	path       string
	wantTask   bool
	wantHTML   bool
	wantStatus int
}

// TestCandidateAccessRouteDispatch keeps candidate landing and API off the Task handler.
func TestCandidateAccessRouteDispatch(t *testing.T) {
	tests := []candidateRouteCase{
		{name: "invitation landing", method: http.MethodGet, path: "/candidate/access?token=secret", wantHTML: true, wantStatus: http.StatusOK},
		{name: "candidate exchange stays on access", method: http.MethodPost, path: "/candidate/access/exchange", wantStatus: http.StatusBadRequest},
		{name: "candidate session stays on access", method: http.MethodGet, path: "/candidate/session", wantStatus: http.StatusBadRequest},
		{name: "admin candidates stay on access", method: http.MethodPost, path: "/admin/candidates", wantStatus: http.StatusBadRequest},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			task := &recordedTaskHandler{}
			router := appruntime.Router{
				Task:    task,
				Access:  accesshandler.Handler{Origin: "http://localhost:8080", Limiter: &accesshandler.LoginLimiter{}, ExchangeLimiter: &accesshandler.LoginLimiter{}},
				Landing: candidatelanding.Handler{},
			}
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(tt.method, tt.path, strings.NewReader("{}"))
			if tt.method != http.MethodGet {
				req.Header.Set("Content-Type", "application/json")
			}
			router.ServeHTTP(rec, req)
			require.Equal(t, tt.wantTask, task.called.Load())
			if tt.wantHTML {
				require.Equal(t, tt.wantStatus, rec.Code)
				require.Contains(t, rec.Header().Get("Content-Type"), "text/html")
			} else {
				require.Equal(t, tt.wantStatus, rec.Code)
			}
		})
	}
}

// configurationRouteCase names one composed-router configuration dispatch outcome.
type configurationRouteCase struct {
	name       string
	method     string
	path       string
	wantConfig bool
	wantStatus int
}

// TestConfigurationRouteDispatch sends agent-templates and agent-instances ahead of the access catch-all.
func TestConfigurationRouteDispatch(t *testing.T) {
	tests := []configurationRouteCase{
		{name: "templates", method: http.MethodGet, path: "/admin/agent-templates", wantConfig: true, wantStatus: http.StatusOK},
		{name: "instances", method: http.MethodPost, path: "/admin/agent-instances", wantConfig: true, wantStatus: http.StatusOK},
		{name: "instance draft", method: http.MethodPut, path: "/admin/agent-instances/aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaa8/draft", wantConfig: true, wantStatus: http.StatusOK},
		{name: "sessions stay on access", method: http.MethodPost, path: "/admin/sessions", wantStatus: http.StatusBadRequest},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			task := &recordedTaskHandler{}
			config := &recordedConfigHandler{}
			router := appruntime.Router{
				Task:          task,
				Access:        accesshandler.Handler{Origin: "http://localhost:8080", Limiter: &accesshandler.LoginLimiter{}, ExchangeLimiter: &accesshandler.LoginLimiter{}},
				Landing:       candidatelanding.Handler{},
				Configuration: config,
			}
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(tt.method, tt.path, strings.NewReader("{}"))
			if tt.method != http.MethodGet {
				req.Header.Set("Content-Type", "application/json")
			}
			router.ServeHTTP(rec, req)
			require.False(t, task.called.Load())
			require.Equal(t, tt.wantConfig, config.called.Load())
			require.Equal(t, tt.wantStatus, rec.Code)
		})
	}
}

// interviewRouteCase names one composed-router candidate interview dispatch outcome.
type interviewRouteCase struct {
	name       string
	path       string
	wantJSON   bool
	wantPage   bool
	wantAccess bool
}

// recordedHandler records which composed handler a request reached.
type recordedHandler struct{ called atomic.Bool }

// ServeHTTP records the request and returns an empty success.
func (h *recordedHandler) ServeHTTP(w http.ResponseWriter, _ *http.Request) {
	h.called.Store(true)
	w.WriteHeader(http.StatusNoContent)
}

// TestCandidateInterviewRouteDispatch sends the interview paths ahead of the access catch-all and leaves access paths alone.
func TestCandidateInterviewRouteDispatch(t *testing.T) {
	tests := []interviewRouteCase{
		{name: "interview read", path: "/candidate/interview", wantJSON: true},
		{name: "consent accept", path: "/candidate/consents", wantJSON: true},
		{name: "page shell", path: "/candidate/interview/page", wantPage: true},
		{name: "exchange stays on access", path: "/candidate/access/exchange", wantAccess: true},
		{name: "session stays on access", path: "/candidate/session", wantAccess: true},
		{name: "unmounted execution route stays on access", path: "/candidate/executions", wantAccess: true},
		{name: "similar prefix is not the interview", path: "/candidate/interviews", wantAccess: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			task, jsonHandler, page, access := &recordedTaskHandler{}, &recordedHandler{}, &recordedHandler{}, &recordedHandler{}
			router := appruntime.Router{Task: task, Access: access, Interview: jsonHandler, InterviewPage: page}

			router.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, tt.path, nil))

			require.False(t, task.called.Load())
			require.Equal(t, tt.wantJSON, jsonHandler.called.Load())
			require.Equal(t, tt.wantPage, page.called.Load())
			require.Equal(t, tt.wantAccess, access.called.Load())
		})
	}
}
