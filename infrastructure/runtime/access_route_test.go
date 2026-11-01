package runtime_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/rfanazhari/amer-prototype/delivery/http/candidatelanding"
	"github.com/rfanazhari/amer-prototype/delivery/http/handler/accesshandler"
	appruntime "github.com/rfanazhari/amer-prototype/infrastructure/runtime"
	"github.com/stretchr/testify/require"
)

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
