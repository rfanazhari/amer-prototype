package candidatelanding_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/rfanazhari/amer-prototype/delivery/http/candidatelanding"
	"github.com/rfanazhari/amer-prototype/pkg/errs"
	"github.com/stretchr/testify/require"
)

// landingDocumentCase names one GET /candidate/access HTML assertion.
type landingDocumentCase struct {
	name     string
	contains []string
	absent   []string
}

// landingMethodCase names one non-GET landing request that must not exchange.
type landingMethodCase struct {
	name   string
	method string
	status int
	code   string
}

// TestInvitationLandingRemovesTokenBeforeExternalWork covers history cleanup, referrer policy and GET refusal.
func TestInvitationLandingRemovesTokenBeforeExternalWork(t *testing.T) {
	tests := []landingDocumentCase{
		{
			name: "strips token from history before network work",
			contains: []string{
				"history.replaceState",
				"no-referrer",
				"no-store",
				"POST",
				"/candidate/access/exchange",
				"credentials",
				"include",
			},
			absent: []string{"consent", "interview-runtime", "/voice-sessions", "OPENAI_API_KEY"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/candidate/access?token=invitation-secret", nil)
			candidatelanding.Handler{}.ServeHTTP(rec, req)
			require.Equal(t, http.StatusOK, rec.Code)
			require.Contains(t, rec.Header().Get("Content-Type"), "text/html")
			require.Equal(t, "no-store", rec.Header().Get("Cache-Control"))
			require.Equal(t, "no-referrer", rec.Header().Get("Referrer-Policy"))
			body := rec.Body.String()
			scriptStart := strings.Index(body, "<script>")
			require.GreaterOrEqual(t, scriptStart, 0)
			script := body[scriptStart:]
			replaceAt := strings.Index(script, "history.replaceState")
			fetchAt := strings.Index(script, "/candidate/access/exchange")
			require.GreaterOrEqual(t, replaceAt, 0)
			require.GreaterOrEqual(t, fetchAt, 0)
			require.Less(t, replaceAt, fetchAt)
			for _, needle := range tt.contains {
				require.Contains(t, body, needle)
			}
			for _, needle := range tt.absent {
				require.NotContains(t, strings.ToLower(body), needle)
			}
		})
	}
}

// TestInvitationLandingRejectsNonGet prevents prefetch-style methods from exchanging.
func TestInvitationLandingRejectsNonGet(t *testing.T) {
	tests := []landingMethodCase{
		{name: "post landing is not exchange", method: http.MethodPost, status: http.StatusMethodNotAllowed, code: errs.MethodNotAllowed},
		{name: "head landing is not exchange", method: http.MethodHead, status: http.StatusMethodNotAllowed, code: errs.MethodNotAllowed},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(tt.method, "/candidate/access?token=invitation-secret", nil)
			candidatelanding.Handler{}.ServeHTTP(rec, req)
			require.Equal(t, tt.status, rec.Code)
			if tt.method == http.MethodHead {
				return
			}
			require.Contains(t, rec.Body.String(), tt.code)
			require.NotContains(t, rec.Body.String(), "Set-Cookie")
		})
	}
}
