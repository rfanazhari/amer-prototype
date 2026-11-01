package agentconfighandler_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/rfanazhari/amer-prototype/pkg/errs"
	"github.com/stretchr/testify/require"
)

// routingHTTPCase names one method or path envelope outcome.
type routingHTTPCase struct {
	name       string
	method     string
	path       string
	wantStatus int
	wantCode   string
}

// TestConfigurationRoutingEnvelope rejects unknown configuration paths and unsupported methods.
func TestConfigurationRoutingEnvelope(t *testing.T) {
	tests := []routingHTTPCase{
		{name: "unknown configuration collection", method: http.MethodGet, path: "/admin/agent-unknown", wantStatus: http.StatusNotFound, wantCode: errs.NotFound},
		{name: "wrong catalog method", method: http.MethodPost, path: "/admin/agent-templates", wantStatus: http.StatusMethodNotAllowed, wantCode: errs.MethodNotAllowed},
		{name: "wrong instance collection method", method: http.MethodGet, path: "/admin/agent-instances", wantStatus: http.StatusMethodNotAllowed, wantCode: errs.MethodNotAllowed},
		{name: "wrong draft method", method: http.MethodPost, path: "/admin/agent-instances/" + instanceID + "/draft", wantStatus: http.StatusMethodNotAllowed, wantCode: errs.MethodNotAllowed},
		{name: "access path is not configuration", method: http.MethodPost, path: "/admin/sessions", wantStatus: http.StatusNotFound, wantCode: errs.NotFound},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler, _, _, _, _, _, _, _, _, _ := newHandler(t, httpOrigin, false)
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, request(tt.method, tt.path, httpOrigin, "{}", "", adminCookie(false, sessionTok), nil))
			require.Equal(t, tt.wantStatus, rec.Code)
			view := decodeEnvelope(t, rec)
			require.Equal(t, "error", view.Status)
			require.Equal(t, tt.wantCode, view.Error.Code)
		})
	}
}
