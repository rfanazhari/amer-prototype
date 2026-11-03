package interviewreporthandler_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/rfanazhari/amer-prototype/application/service"
	"github.com/rfanazhari/amer-prototype/domain/enum"
)

// guardCase combines invalid input with authentication and authorization failures.
type guardCase struct {
	name       string
	cookie     string
	sessionErr error
	admin      bool
	noCSRF     bool
	badCSRF    bool
	badOrigin  bool
	status     int
	code       string
}

// TestReportPrecedence rejects unauthorized requests before malformed ids, bodies and operations.
func TestReportPrecedence(t *testing.T) {
	tests := []guardCase{
		{name: "missing cookie", status: 401, code: "IRP_UNAUTHENTICATED"},
		{name: "candidate cookie", cookie: "__Host-ia-candidate=candidate", status: 401, code: "IRP_UNAUTHENTICATED"},
		{name: "wrong transport cookie", cookie: "ia-admin=session-secret", status: 401, code: "IRP_UNAUTHENTICATED"},
		{name: "expired revoked disabled or unbound", cookie: "__Host-ia-admin=session-secret", sessionErr: service.ErrSessionUnbound, status: 401, code: "IRP_UNAUTHENTICATED"},
		{name: "admin before parsing", cookie: "__Host-ia-admin=session-secret", admin: true, status: 403, code: "IRP_FORBIDDEN"},
		{name: "missing csrf before parsing", cookie: "__Host-ia-admin=session-secret", noCSRF: true, status: 403, code: "IRP_FORBIDDEN"},
		{name: "bad csrf before parsing", cookie: "__Host-ia-admin=session-secret", badCSRF: true, status: 403, code: "IRP_FORBIDDEN"},
		{name: "foreign origin", cookie: "__Host-ia-admin=session-secret", badOrigin: true, status: 403, code: "IRP_FORBIDDEN"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newFixture(t)
			r := request(http.MethodPost, "/admin/interview-assignments/bad/report-generations", `{"salaryCeiling":100}`)
			r.Header.Del("Cookie")
			if tt.cookie != "" {
				r.Header.Set("Cookie", tt.cookie)
			}
			if tt.cookie == "__Host-ia-admin=session-secret" {
				role := enum.HR
				if tt.admin {
					role = enum.Admin
				}
				f.session(t, role, tt.sessionErr)
				if tt.sessionErr == nil && !tt.admin && !tt.noCSRF && !tt.badOrigin {
					f.allowCSRF(t, service.ErrCSRFMismatch)
				}
			}
			if tt.noCSRF {
				r.Header.Del("X-CSRF-Token")
			}
			if tt.badOrigin {
				r.Header.Set("Origin", "https://foreign.test")
			}
			rec := httptest.NewRecorder()
			f.h.ServeHTTP(rec, r)
			assertError(t, rec, tt.status, tt.code)
		})
	}
}

// inputCase defines transport rejection without invoking an application operation.
type inputCase struct {
	name   string
	method string
	path   string
	body   string
	change func(*http.Request)
	status int
	code   string
}

// TestReportInput enforces empty JSON objects, exact paths, key rules and body-free GETs.
func TestReportInput(t *testing.T) {
	tests := []inputCase{
		{name: "missing body", method: "POST", path: postPath, status: 400, code: "INVALID_INPUT"},
		{name: "null body", method: "POST", path: postPath, body: "null", status: 400, code: "INVALID_INPUT"},
		{name: "array", method: "POST", path: postPath, body: "[]", status: 400, code: "INVALID_INPUT"},
		{name: "unknown verdict", method: "POST", path: postPath, body: `{"decision":"hire"}`, status: 400, code: "INVALID_INPUT"},
		{name: "unknown salary", method: "POST", path: postPath, body: `{"salaryCeiling":1}`, status: 400, code: "INVALID_INPUT"},
		{name: "duplicate field", method: "POST", path: postPath, body: `{"x":1,"x":2}`, status: 400, code: "INVALID_INPUT"},
		{name: "trailing JSON", method: "POST", path: postPath, body: "{} {}", status: 400, code: "INVALID_INPUT"},
		{name: "oversized", method: "POST", path: postPath, body: "{" + strings.Repeat(" ", 1<<20) + "}", status: 400, code: "INVALID_INPUT"},
		{name: "content type", method: "POST", path: postPath, body: "{}", change: func(r *http.Request) { r.Header.Set("Content-Type", "text/plain") }, status: 400, code: "INVALID_INPUT"},
		{name: "missing key", method: "POST", path: postPath, body: "{}", change: func(r *http.Request) { r.Header.Del("Idempotency-Key") }, status: 400, code: "INVALID_INPUT"},
		{name: "duplicate key", method: "POST", path: postPath, body: "{}", change: func(r *http.Request) { r.Header.Add("Idempotency-Key", "report-1") }, status: 400, code: "INVALID_INPUT"},
		{name: "key no trim", method: "POST", path: postPath, body: "{}", change: func(r *http.Request) { r.Header.Set("Idempotency-Key", " report-1") }, status: 400, code: "INVALID_INPUT"},
		{name: "long key", method: "POST", path: postPath, body: "{}", change: func(r *http.Request) { r.Header.Set("Idempotency-Key", strings.Repeat("a", 129)) }, status: 400, code: "INVALID_INPUT"},
		{name: "key punctuation start", method: "POST", path: postPath, body: "{}", change: func(r *http.Request) { r.Header.Set("Idempotency-Key", ".bad") }, status: 400, code: "INVALID_INPUT"},
		{name: "malformed UUID", method: "GET", path: "/admin/interview-assignments/bad/report", status: 400, code: "INVALID_INPUT"},
		{name: "nil UUID", method: "GET", path: "/admin/interview-assignments/00000000-0000-0000-0000-000000000000/report", status: 400, code: "INVALID_INPUT"},
		{name: "query", method: "GET", path: readPath + "?salaryCeiling=1", status: 400, code: "INVALID_INPUT"},
		{name: "GET body", method: "GET", path: readPath, body: "{}", status: 400, code: "INVALID_INPUT"},
		{name: "GET missing origin", method: "GET", path: readPath, change: func(r *http.Request) { r.Header.Del("Origin") }, status: 403, code: "IRP_FORBIDDEN"},
		{name: "wrong method", method: "DELETE", path: readPath, status: 405, code: "METHOD_NOT_ALLOWED"},
		{name: "wrong route", method: "GET", path: readPath + "/extra", status: 404, code: "NOT_FOUND"},
		{name: "candidate path", method: "GET", path: "/candidate/interview-assignments/" + assignmentID + "/report", status: 404, code: "NOT_FOUND"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newFixture(t)
			r := request(tt.method, tt.path, tt.body)
			if tt.change != nil {
				tt.change(r)
			}
			if tt.status != 404 && tt.status != 405 {
				f.session(t, enum.HR, nil)
				if tt.method == "POST" {
					f.allowCSRF(t, nil)
				}
			}
			rec := httptest.NewRecorder()
			f.h.ServeHTTP(rec, r)
			assertError(t, rec, tt.status, tt.code)
		})
	}
}
