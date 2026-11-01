package livevoice_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// guardCase describes untrusted browser requests that must not reach provider setup.
type guardCase struct {
	name   string
	host   string
	origin string
}

// TestRequestGuards proves host and origin rejection happens before paid provider work.
func TestRequestGuards(t *testing.T) {
	for _, tt := range []guardCase{
		{name: "foreign host", host: "attacker.example", origin: "http://127.0.0.1:1234"},
		{name: "foreign origin", host: "127.0.0.1:1234", origin: "https://attacker.example"},
		{name: "missing origin", host: "127.0.0.1:1234"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			p := &probe{origin: "http://127.0.0.1:1234", counts: make(map[string]int)}
			request := httptest.NewRequest(http.MethodPost, "http://"+tt.host+"/setup", strings.NewReader(`{"sdp":"offer"}`))
			request.Header.Set("Origin", tt.origin)
			response := httptest.NewRecorder()
			p.ServeHTTP(response, request)
			require.Equal(t, http.StatusForbidden, response.Code)
			require.Zero(t, p.attempts)
		})
	}
}

// TestLocalGenerationFence proves only harness-local event isolation, not durable platform fencing.
func TestLocalGenerationFence(t *testing.T) {
	p := &probe{generation: 2, counts: make(map[string]int)}
	p.observe(1, "session.started")
	p.observe(2, "session.started")
	require.Equal(t, 1, p.counts["session.started"])
	require.Equal(t, 1, p.counts["probe.stale_event_rejected"])
}
