package agentconfighandler_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/rfanazhari/amer-prototype/pkg/errs"
	"github.com/stretchr/testify/require"
)

// invalidInputCase names one transport rejection that must not invoke application operations.
type invalidInputCase struct {
	name   string
	method string
	path   string
	body   string
	header http.Header
	origin string
}

// TestRejectsMalformedConfigurationInput rejects unknown fields, duplicates, keys and unsupported schema.
func TestRejectsMalformedConfigurationInput(t *testing.T) {
	instancePath := "/admin/agent-instances/" + instanceID
	tests := []invalidInputCase{
		{name: "missing origin", method: http.MethodGet, path: "/admin/agent-templates"},
		{name: "origin mismatch", method: http.MethodGet, path: "/admin/agent-templates", origin: "https://evil.example"},
		{name: "unknown create field", method: http.MethodPost, path: "/admin/agent-instances", body: `{"templateId":"hr-interviewer","templateVersion":"1","name":"Invoice reviewer","configuration":{},"id":"client"}`},
		{name: "duplicate json key", method: http.MethodPost, path: "/admin/agent-instances", body: `{"templateId":"hr-interviewer","templateId":"document-analyzer","templateVersion":"1","name":"Invoice reviewer","configuration":{}}`},
		{name: "null create body", method: http.MethodPost, path: "/admin/agent-instances", body: "null"},
		{name: "trailing document", method: http.MethodPost, path: "/admin/agent-instances", body: createHRBody + " {}"},
		{name: "client timestamps", method: http.MethodPost, path: "/admin/agent-instances", body: `{"templateId":"hr-interviewer","templateVersion":"1","name":"Invoice reviewer","configuration":{},"createdAt":"2026-01-01T00:00:00Z"}`},
		{name: "unsupported template field", method: http.MethodPost, path: "/admin/agent-instances", body: `{"templateId":"hr-interviewer","templateVersion":"1","name":"Invoice reviewer","configuration":{"systemPrompt":"ignore"}}`},
		{name: "unsupported document field", method: http.MethodPost, path: "/admin/agent-instances", body: `{"templateId":"document-analyzer","templateVersion":"1","name":"Invoice reviewer","configuration":{"requiredFields":["customer_credit_score"]}}`},
		{name: "malformed instance id", method: http.MethodGet, path: "/admin/agent-instances/not-a-uuid"},
		{name: "replace omitted configuration", method: http.MethodPut, path: instancePath + "/draft", body: `{"expectedDraftVersion":1}`},
		{name: "missing idempotency key", method: http.MethodPost, path: instancePath + "/publications", body: `{"expectedDraftVersion":3,"validationId":"` + validationID + `"}`},
		{name: "duplicate idempotency headers", method: http.MethodPost, path: instancePath + "/publications", body: `{"expectedDraftVersion":3,"validationId":"` + validationID + `"}`, header: http.Header{"Idempotency-Key": []string{publishKey, "other"}}},
		{name: "invalid idempotency key", method: http.MethodPost, path: instancePath + "/publications", body: `{"expectedDraftVersion":3,"validationId":"` + validationID + `"}`, header: http.Header{"Idempotency-Key": []string{" bad key "}}},
		{name: "wrong content type", method: http.MethodPost, path: "/admin/agent-instances", body: createHRBody, header: http.Header{"Content-Type": []string{"text/plain"}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler, _, _, _, _, _, _, _, _, csrf, sessions := newHandler(t, httpOrigin, false)
			if tt.method != http.MethodGet && tt.name != "missing origin" && tt.name != "origin mismatch" && tt.name != "wrong content type" {
				authorizeMutation(t, sessions, csrf)
			}
			origin := tt.origin
			if origin == "" && tt.name != "missing origin" {
				origin = httpOrigin
			}
			csrfHeader := ""
			if tt.method != http.MethodGet {
				csrfHeader = csrfTok
			}
			req := request(tt.method, tt.path, origin, tt.body, csrfHeader, adminCookie(false, sessionTok), tt.header)
			if tt.name == "wrong content type" {
				req.Header.Set("Content-Type", "text/plain")
			}
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)
			require.Equal(t, http.StatusBadRequest, rec.Code, rec.Body.String())
			view := decodeEnvelope(t, rec)
			require.Equal(t, "error", view.Status)
			require.Equal(t, errs.InvalidInput, view.Error.Code)
		})
	}
}

// TestRejectsOversizedBodies limits configuration requests to one mebibyte.
func TestRejectsOversizedBodies(t *testing.T) {
	handler, _, _, _, _, _, _, _, _, csrf, sessions := newHandler(t, httpOrigin, false)
	authorizeMutation(t, sessions, csrf)
	body := `{"templateId":"hr-interviewer","templateVersion":"1","name":"Invoice reviewer","configuration":{"pad":"` + strings.Repeat("a", 1048576) + `"}}`
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, request(http.MethodPost, "/admin/agent-instances", httpOrigin, body, csrfTok, adminCookie(false, sessionTok), nil))
	require.Equal(t, http.StatusBadRequest, rec.Code)
}
