package candidateinterview_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/rfanazhari/amer-prototype/delivery/http/candidateinterview"
	"github.com/rfanazhari/amer-prototype/pkg/errs"
	"github.com/stretchr/testify/require"
)

// TestInterviewPageShellDocument verifies the HTML headers and that the shell talks only to the mounted JSON routes.
func TestInterviewPageShellDocument(t *testing.T) {
	rec := httptest.NewRecorder()

	candidateinterview.Handler{}.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/candidate/interview/page", nil))

	require.Equal(t, http.StatusOK, rec.Code)
	require.Contains(t, rec.Header().Get("Content-Type"), "text/html")
	require.Equal(t, "no-store", rec.Header().Get("Cache-Control"))
	require.Equal(t, "no-referrer", rec.Header().Get("Referrer-Policy"))
	body := rec.Body.String()
	for _, needle := range []string{"/candidate/session", "/candidate/interview", "/candidate/consents", "X-CSRF-Token", "credentials", "include"} {
		require.Contains(t, body, needle)
	}
	lower := strings.ToLower(body)
	for _, absent := range []string{"/candidate/executions", "report", "getusermedia", "rtcpeerconnection", "salary", "/voice-session"} {
		require.NotContains(t, lower, absent)
	}
}

// TestInterviewPageShellShowsAccessErrorBeforeNotice verifies a missing cookie renders the access-error state, not the notice.
func TestInterviewPageShellShowsAccessErrorBeforeNotice(t *testing.T) {
	rec := httptest.NewRecorder()

	candidateinterview.Handler{}.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/candidate/interview/page", nil))

	body := rec.Body.String()
	require.Contains(t, body, "<script>")
	script := body[strings.Index(body, "<script>"):]
	sessionAt := strings.Index(script, "/candidate/session")
	interviewAt := strings.Index(script, `"/candidate/interview"`)
	noticeAt := strings.Index(script, "statement")
	require.GreaterOrEqual(t, sessionAt, 0)
	require.Greater(t, interviewAt, sessionAt, "the session is read before the interview")
	require.Greater(t, noticeAt, interviewAt, "the notice is rendered only from the interview response")
	require.Contains(t, script, "401")
	require.Contains(t, script, "Access is no longer available")
}

// TestInterviewPageShellRejectsOtherMethods verifies only GET serves the document.
func TestInterviewPageShellRejectsOtherMethods(t *testing.T) {
	for _, method := range []string{http.MethodPost, http.MethodPut, http.MethodDelete} {
		t.Run(method, func(t *testing.T) {
			rec := httptest.NewRecorder()

			candidateinterview.Handler{}.ServeHTTP(rec, httptest.NewRequest(method, "/candidate/interview/page", nil))

			require.Equal(t, http.StatusMethodNotAllowed, rec.Code)
			require.Contains(t, rec.Body.String(), errs.MethodNotAllowed)
		})
	}
}
