package taskhandler_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/rfanazhari/amer-prototype/delivery/http/handler/taskhandler"
	"github.com/stretchr/testify/require"
)

// routingCase defines an HTTP readiness or routing envelope scenario.
type routingCase struct {
	name, method, path string
	status             int
	result             string
}

// TestHealthAndRoutingEnvelope checks unauthenticated health and standard envelopes for routing and method errors.
func TestHealthAndRoutingEnvelope(t *testing.T) {
	tests := []routingCase{
		{"health", http.MethodGet, "/health", http.StatusOK, "success"},
		{"unknown route", http.MethodGet, "/missing", http.StatusNotFound, "error"},
		{"wrong method", http.MethodPost, "/health", http.StatusMethodNotAllowed, "error"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			req := httptest.NewRequest(tt.method, tt.path, nil)
			taskhandler.Handler{}.ServeHTTP(w, req)
			require.Equal(t, tt.status, w.Code)
			var body map[string]json.RawMessage
			require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
			var status string
			require.NoError(t, json.Unmarshal(body["status"], &status))
			require.Equal(t, tt.result, status)
			require.JSONEq(t, `{}`, string(body["data"]))
		})
	}
}

// invalidBodyCase defines a request that must fail transport validation before application execution.
type invalidBodyCase struct{ name, method, path, body string }

// TestRejectsNonObjectBodies rejects null action bodies, omitted rescheduling fields and unsupported date precision.
func TestRejectsNonObjectBodies(t *testing.T) {
	tests := []invalidBodyCase{
		{"null create", http.MethodPost, "/tasks", "null"},
		{"null start", http.MethodPost, "/tasks/9d96c891-8118-46d4-92d2-79da2337faba/start", "null"},
		{"absent due date", http.MethodPatch, "/tasks/9d96c891-8118-46d4-92d2-79da2337faba/due-date", "{}"},
		{"excess precision", http.MethodPost, "/tasks", `{"title":"task","dueDate":"2030-01-01T00:00:00.0001Z"}`},
		{"excess comma precision", http.MethodPost, "/tasks", `{"title":"task","dueDate":"2030-01-01T00:00:00,0001Z"}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			taskhandler.Handler{}.ServeHTTP(w, httptest.NewRequest(tt.method, tt.path, strings.NewReader(tt.body)))
			require.Equal(t, http.StatusBadRequest, w.Code, w.Body.String())
		})
	}
}

// TestRejectsUnexpectedClientFields rejects client-owned system facts, unknown fields and multiple JSON documents.
func TestRejectsUnexpectedClientFields(t *testing.T) {
	tests := []invalidBodyCase{
		{"client id", "POST", "/tasks", `{"title":"task","id":"client-id"}`},
		{"client status", "POST", "/tasks", `{"title":"task","status":"done"}`},
		{"client timestamp", "POST", "/tasks", `{"title":"task","createdAt":"2026-01-01T00:00:00Z"}`},
		{"start fields", "POST", "/tasks/9d96c891-8118-46d4-92d2-79da2337faba/start", `{"title":"unexpected"}`},
		{"multiple documents", "POST", "/tasks", `{"title":"task"} {}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			taskhandler.Handler{}.ServeHTTP(w, httptest.NewRequest(tt.method, tt.path, strings.NewReader(tt.body)))
			require.Equal(t, http.StatusBadRequest, w.Code)
		})
	}
}
