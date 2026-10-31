// Package livevoice_test provides an opt-in, loopback-only manual provider probe.
package livevoice_test

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/SALT-Indonesia/salt-pkg/clientmanager"
	"github.com/SALT-Indonesia/salt-pkg/logmanager"
	"github.com/rfanazhari/amer-prototype/infrastructure/service/liveprobe"
	"github.com/stretchr/testify/require"
	"golang.org/x/net/websocket"
)

// page is served only by the explicitly enabled local test harness.
//
//go:embed page.html
var page string

// probe owns one bounded session and payload-free observation counts.
type probe struct {
	mu         sync.Mutex
	conn       *websocket.Conn
	counts     map[string]int
	generation int
	attempts   int
	key        string
	origin     string
	logs       *logmanager.Application
	ctx        context.Context
	done       chan struct{}
	finish     sync.Once
	readers    sync.WaitGroup
}

// observe records only event names, fencing events from replaced connections.
func (p *probe) observe(generation int, kind string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if generation != p.generation {
		p.counts["probe.stale_event_rejected"]++
		return
	}
	if len(p.counts) < 256 {
		p.counts[kind]++
	}
}

// closeSession fences the connection before sending a bounded best-effort close command.
func (p *probe) closeSession() {
	p.mu.Lock()
	c := p.conn
	p.conn = nil
	p.generation++
	p.mu.Unlock()
	if c != nil {
		_ = c.SetWriteDeadline(time.Now().Add(time.Second))
		err := websocket.Message.Send(c, `{"type":"session.close"}`)
		p.mu.Lock()
		if err != nil {
			p.counts["probe.close_send_failed"]++
		} else {
			p.counts["probe.close_sent_unconfirmed"]++
		}
		p.mu.Unlock()
		_ = c.Close()
	}
}

// read observes sideband event names without writing transcript, audio, or provider payloads.
func (p *probe) read(c *websocket.Conn, generation int) {
	defer p.readers.Done()
	defer func() { _ = c.Close() }()
	for {
		var body []byte
		if websocket.Message.Receive(c, &body) != nil {
			p.observe(generation, "probe.sideband_ended")
			return
		}
		kind, err := liveprobe.Shape(body)
		if err != nil {
			p.observe(generation, "probe.unrecognized_event")
			continue
		}
		p.observe(generation, kind)
	}
}

// setup creates one provider session then attaches the backend before returning the SDP answer.
func (p *probe) setup(w http.ResponseWriter, r *http.Request) {
	p.mu.Lock()
	if p.conn != nil || p.attempts >= 4 {
		p.mu.Unlock()
		http.Error(w, "close current session or restart harness", http.StatusConflict)
		return
	}
	p.attempts++
	p.generation++
	generation := p.generation
	p.mu.Unlock()
	var input map[string]string
	if json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&input) != nil || strings.TrimSpace(input["sdp"]) == "" {
		http.Error(w, "invalid offer", http.StatusBadRequest)
		return
	}
	ctx, cancel := context.WithTimeout(p.ctx, 30*time.Second)
	defer cancel()
	txn := p.logs.Start("", "synthetic live probe", logmanager.TxnTypeOther)
	defer txn.End()
	body := map[string]any{
		"session":   map[string]any{"model": "gpt-live-1", "instructions": "Speak Indonesian. This is a synthetic interview feasibility test. Ask about a fictional Go project. Do not request personal data. Do not score or make hiring decisions."},
		"transport": map[string]string{"type": "webrtc", "sdp": input["sdp"]},
	}
	response, err := clientmanager.CallBytes(txn.ToContext(ctx), "https://api.openai.com/v1/live/sessions",
		clientmanager.WithMethod(http.MethodPost), clientmanager.WithAuth(clientmanager.AuthBearer(p.key)),
		clientmanager.WithRequestBody(body), clientmanager.WithTimeout(30*time.Second), clientmanager.WithMaxResponseBytes(1<<20))
	if err != nil || response == nil {
		p.observe(generation, "probe.setup_transport_failed")
		http.Error(w, "provider setup unavailable", http.StatusServiceUnavailable)
		return
	}
	if !response.IsSuccess() {
		p.observe(generation, fmt.Sprintf("probe.setup_http_%d", response.StatusCode))
		http.Error(w, "provider rejected setup", http.StatusServiceUnavailable)
		return
	}
	setup, err := liveprobe.DecodeSetup(response.Body)
	if err != nil {
		p.observe(generation, "probe.setup_contract_mismatch")
		http.Error(w, "provider setup contract mismatch", http.StatusBadGateway)
		return
	}
	cfg, err := websocket.NewConfig("wss://api.openai.com/v1/live/sessions/"+setup.SessionID+"/attach", "https://api.openai.com")
	if err != nil {
		http.Error(w, "sideband configuration failed", http.StatusInternalServerError)
		return
	}
	cfg.Header.Set("Authorization", "Bearer "+p.key)
	c, err := cfg.DialContext(ctx)
	if err != nil {
		p.observe(generation, "probe.attach_failed_cleanup_unconfirmed")
		http.Error(w, "sideband attach failed; provider cleanup unconfirmed", http.StatusBadGateway)
		return
	}
	c.MaxPayloadBytes = 2 << 20
	_ = c.SetDeadline(time.Now().Add(3 * time.Minute))
	p.mu.Lock()
	if p.generation != generation {
		p.mu.Unlock()
		_ = c.Close()
		http.Error(w, "superseded", http.StatusConflict)
		return
	}
	p.conn = c
	p.readers.Add(1)
	p.mu.Unlock()
	go p.read(c, generation)
	p.observe(generation, "probe.sideband_attached")
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"sdp": setup.SDP, "attempt": generation})
}

// ServeHTTP restricts browser mutations to the loopback origin and returns sanitized observations.
func (p *probe) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Referrer-Policy", "no-referrer")
	if "http://"+r.Host != p.origin {
		http.Error(w, "host rejected", http.StatusForbidden)
		return
	}
	if r.Method == http.MethodPost && r.Header.Get("Origin") != p.origin {
		http.Error(w, "origin rejected", http.StatusForbidden)
		return
	}
	switch r.Method + " " + r.URL.Path {
	case "GET /":
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(page))
	case "GET /observations":
		p.mu.Lock()
		defer p.mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(p.counts)
	case "POST /setup":
		p.setup(w, r)
	case "POST /close":
		p.closeSession()
		w.WriteHeader(http.StatusNoContent)
	case "POST /finish":
		p.finish.Do(func() { close(p.done) })
		w.WriteHeader(http.StatusNoContent)
	default:
		http.NotFound(w, r)
	}
}

// TestManualProbe hosts a bounded, explicitly opted-in browser/provider experiment; it is not an MVP acceptance test.
func TestManualProbe(t *testing.T) {
	if os.Getenv("LIVE_PROBE") != "1" {
		t.Skip("set LIVE_PROBE=1 and OPENAI_API_KEY for manual browser probe")
	}
	key := os.Getenv("OPENAI_API_KEY")
	require.NotEmpty(t, key, "OPENAI_API_KEY must be supplied through environment")
	logs := logmanager.NewApplication(logmanager.WithAppName("CORE-000"), logmanager.WithEnvironment("production"), logmanager.WithSkipHeaders(), logmanager.WithMaskingConfig([]logmanager.MaskingConfig{
		{FieldPattern: "authorization", Type: logmanager.FullMask}, {FieldPattern: "sdp", Type: logmanager.FullMask},
		{FieldPattern: "instructions", Type: logmanager.FullMask}, {FieldPattern: "id", Type: logmanager.FullMask},
	}))
	ctx, cancel := context.WithTimeout(context.Background(), 12*time.Minute)
	defer cancel()
	p := &probe{counts: make(map[string]int), key: key, logs: logs, ctx: ctx, done: make(chan struct{})}
	server := httptest.NewUnstartedServer(p)
	p.origin = "http://" + server.Listener.Addr().String()
	server.Config.ReadHeaderTimeout = 5 * time.Second
	server.Config.WriteTimeout = 35 * time.Second
	server.Start()
	defer server.Close()
	t.Logf("Open %s ; synthetic speech only; press Finish within 12 minutes", server.URL)
	select {
	case <-p.done:
	case <-ctx.Done():
		t.Error("manual probe deadline elapsed")
	}
	cancel()
	p.closeSession()
	server.Close()
	p.readers.Wait()
	p.mu.Lock()
	defer p.mu.Unlock()
	t.Logf("Payload-free observations: %v", p.counts)
	require.Positive(t, p.counts["probe.sideband_attached"], "no provider sideband established; feasibility remains unverified")
	t.Log("Harness completed; event counts do NOT establish transcript finality, durable handoff, recovery, or MVP acceptance")
}
