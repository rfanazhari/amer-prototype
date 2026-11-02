package integration_test

import (
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/SALT-Indonesia/salt-pkg/logmanager"
	"github.com/rfanazhari/amer-prototype/delivery/http/candidateinterview"
	appruntime "github.com/rfanazhari/amer-prototype/infrastructure/runtime"
	"github.com/stretchr/testify/require"
)

// TestCandidatePageDemoPostgres serves the real candidate page over production composition and release-schema
// PostgreSQL so a person or browser tool can drive it. It runs only when CIN_DEMO_HOLD_SECONDS is set.
//
// The access routes are outside this slice, so GET /candidate/session is a stand-in that returns the fixture
// CSRF token, and the fixture session cookie is attached to every request instead of an invitation exchange.
func TestCandidatePageDemoPostgres(t *testing.T) {
	hold, err := strconv.Atoi(os.Getenv("CIN_DEMO_HOLD_SECONDS"))
	if err != nil || hold <= 0 {
		t.Skip("CIN_DEMO_HOLD_SECONDS is not set")
	}
	f := newCandidateLifecycleHTTP(t)
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	origin := "http://" + listener.Addr().String()
	cfg := appruntime.Config{AccessHTTPOrigin: origin, AccessCSRFSecret: strings.Repeat("c", 32)}
	api, _ := appruntime.NewCandidateInterview(f.db, cfg, logmanager.NewApplication(logmanager.WithAppName("cin-demo")))
	csrf := f.csrf
	mux := http.NewServeMux()
	mux.Handle("/candidate/interview/page", candidateinterview.Handler{})
	mux.HandleFunc("/candidate/session", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprintf(w, `{"status":"success","data":{"csrfToken":%q}}`, csrf)
	})
	mux.HandleFunc("/candidate/", func(w http.ResponseWriter, r *http.Request) {
		r.AddCookie(&http.Cookie{Name: "ia-candidate", Value: f.secret})
		api.ServeHTTP(w, r)
	})
	server := httptest.NewUnstartedServer(mux)
	server.Listener = listener
	server.Start()
	t.Cleanup(server.Close)
	t.Logf("CANDIDATE_PAGE_URL=%s/candidate/interview/page", origin)
	fmt.Printf("CANDIDATE_PAGE_URL=%s/candidate/interview/page\n", origin)
	time.Sleep(time.Duration(hold) * time.Second)
}
