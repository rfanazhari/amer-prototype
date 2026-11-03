package interviewvoice_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/SALT-Indonesia/salt-pkg/logmanager"
	"github.com/rfanazhari/amer-prototype/application/service"
	"github.com/rfanazhari/amer-prototype/application/usecase/voicesession"
	"github.com/rfanazhari/amer-prototype/infrastructure/service/interviewvoice"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/net/websocket"
)

const (
	testKey        = "server-key"
	providerSecret = "secret provider dump"
	providerID     = "sess_abc-123"
	answerSDP      = "v=0 answer"
)

// fakeProvider is a local GPT-Live stand-in; it never reaches the public network.
type fakeProvider struct {
	server    *httptest.Server
	calls     atomic.Int32
	status    int
	body      string
	request   map[string]any
	auth      string
	mu        sync.Mutex
	attachKey string
	attachID  string
	push      []string
	hangUp    bool
	received  chan string
}

// newProvider starts a fake whose setup reply and sideband script come from the arguments.
func newProvider(t *testing.T, status int, body string, push ...string) *fakeProvider {
	t.Helper()
	p := &fakeProvider{status: status, body: body, push: push, received: make(chan string, 8)}
	mux := http.NewServeMux()
	mux.HandleFunc("POST /sessions", func(w http.ResponseWriter, r *http.Request) {
		p.calls.Add(1)
		p.auth = r.Header.Get("Authorization")
		assert.NoError(t, json.NewDecoder(r.Body).Decode(&p.request))
		w.WriteHeader(p.status)
		_, err := io.WriteString(w, p.body)
		assert.NoError(t, err)
	})
	mux.Handle("GET /sessions/{id}/attach", websocket.Handler(func(c *websocket.Conn) {
		p.mu.Lock()
		p.attachKey = c.Request().Header.Get("Authorization")
		p.attachID = c.Request().PathValue("id")
		p.mu.Unlock()
		for _, m := range p.push {
			assert.NoError(t, websocket.Message.Send(c, m))
		}
		if p.hangUp {
			return
		}
		for {
			var in string
			if websocket.Message.Receive(c, &in) != nil {
				return
			}
			p.received <- in
		}
	}))
	p.server = httptest.NewServer(mux)
	t.Cleanup(p.server.Close)
	return p
}

// attached returns the credential and provider id of the latest attach under the lock the handler writes with.
func (p *fakeProvider) attached() (key, id string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.attachKey, p.attachID
}

// driver builds a Driver aimed at the fake provider.
func (p *fakeProvider) driver(key string) *interviewvoice.Driver {
	return &interviewvoice.Driver{
		APIKey:      key,
		Model:       "gpt-live-1",
		Voice:       "marin",
		SessionsURL: p.server.URL + "/sessions",
		AttachURL:   "ws" + strings.TrimPrefix(p.server.URL, "http") + "/sessions",
	}
}

// session returns the correlation identity used across calls.
func session() service.InterviewVoiceSession {
	return service.InterviewVoiceSession{ExecutionID: "exec-1", AttemptID: "attempt-1", SessionID: "local-1", ProviderSessionID: providerID}
}

// tracedContext carries the logmanager transaction clientmanager requires.
func tracedContext(t *testing.T) context.Context {
	t.Helper()
	logs := logmanager.NewApplication(
		logmanager.WithAppName("interviewvoice-test"),
		logmanager.WithSkipHeaders(),
		logmanager.WithMaskingConfig([]logmanager.MaskingConfig{{FieldPattern: "authorization", Type: logmanager.FullMask}}),
	)
	txn := logs.Start("", "interview voice", logmanager.TxnTypeOther)
	t.Cleanup(txn.End)
	return txn.ToContext(context.Background())
}

// TestDriverCreate checks retained provider identity, SDP answer, and safe setup failures.
func TestDriverCreate(t *testing.T) {
	// createCase defines the key, provider reply and expected interaction.
	type createCase struct {
		name      string
		key       string
		status    int
		body      string
		closed    bool
		wantCalls int32
		wantErr   bool
	}
	ok := `{"session":{"id":"` + providerID + `"},"transport":{"sdp":"` + answerSDP + `","type":"webrtc"}}`
	tests := []createCase{
		{name: "retains provider session id and answer", key: testKey, status: http.StatusCreated, body: ok, wantCalls: 1},
		{name: "empty key never dials", key: "", status: http.StatusCreated, body: ok, wantCalls: 0, wantErr: true},
		{name: "non-success hides the provider body", key: testKey, status: http.StatusInternalServerError, body: providerSecret, wantCalls: 1, wantErr: true},
		{name: "answer without session id is unavailable", key: testKey, status: http.StatusCreated, body: `{"transport":{"sdp":"` + answerSDP + `","type":"webrtc"}}`, wantCalls: 1, wantErr: true},
		{name: "transport failure", key: testKey, closed: true, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := newProvider(t, tt.status, tt.body)
			d := p.driver(tt.key)
			if tt.closed {
				p.server.Close()
			}
			got, err := d.Create(tracedContext(t), service.InterviewVoiceCreate{
				Session:      service.InterviewVoiceSession{ExecutionID: "exec-1", AttemptID: "attempt-1", SessionID: "local-1"},
				SDP:          "v=0 offer",
				Instructions: "interview in Indonesian",
			})
			assert.Equal(t, tt.wantCalls, p.calls.Load())
			if tt.wantErr {
				require.ErrorIs(t, err, voicesession.ErrSessionUnavailable)
				assert.NotContains(t, err.Error(), providerSecret)
				assert.NotContains(t, err.Error(), testKey)
				assert.Equal(t, service.InterviewVoiceCreated{}, got)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, "Bearer "+testKey, p.auth)
			assert.Equal(t, session(), got.Session)
			assert.Equal(t, answerSDP, got.SDP)
			transport, _ := p.request["transport"].(map[string]any)
			assert.Equal(t, "v=0 offer\r\n", transport["sdp"])
			assert.Equal(t, "webrtc", transport["type"])
			cfg, _ := p.request["session"].(map[string]any)
			assert.Equal(t, "gpt-live-1", cfg["model"])
			assert.Equal(t, true, cfg["store"], "new provider sessions must request recording storage")
			assert.Equal(t, "interview in Indonesian", cfg["instructions"])
		})
	}
}

// TestDriverObserve checks opaque events, disconnects, and cancellation without coverage meaning.
func TestDriverObserve(t *testing.T) {
	// observeCase scripts what the sideband delivers and what Observe must return.
	type observeCase struct {
		name      string
		push      []string
		hangUp    bool
		attach    bool
		cancel    bool
		wantKind  string
		wantType  string
		wantBody  string
		wantErrIs error
	}
	tests := []observeCase{
		{name: "named event stays opaque", push: []string{`{"type":"x.unmapped","n":1}`}, attach: true, wantKind: "event", wantType: "x.unmapped", wantBody: `{"type":"x.unmapped","n":1}`},
		{name: "non-json payload has no event type", push: []string{`not json`}, attach: true, wantKind: "event", wantBody: `not json`},
		{name: "hang up reports disconnected", hangUp: true, attach: true, wantKind: "disconnected"},
		{name: "not attached", attach: false, wantErrIs: interviewvoice.ErrNotAttached},
		{name: "cancelled wait ends", attach: true, cancel: true, wantErrIs: context.Canceled},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := newProvider(t, http.StatusCreated, "", tt.push...)
			p.hangUp = tt.hangUp
			d := p.driver(testKey)
			ctx := context.Background()
			if tt.attach {
				require.NoError(t, d.Attach(ctx, session()))
				t.Cleanup(func() { _ = d.Close(ctx, session()) })
			}
			if tt.cancel {
				var stop context.CancelFunc
				ctx, stop = context.WithTimeout(ctx, 100*time.Millisecond)
				stop()
			}
			got, err := d.Observe(ctx, session())
			if tt.wantErrIs != nil {
				require.ErrorIs(t, err, tt.wantErrIs)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, session(), got.Session)
			assert.Equal(t, tt.wantKind, got.Kind)
			assert.Equal(t, tt.wantType, got.EventType)
			assert.Equal(t, tt.wantBody, string(got.Payload))
		})
	}
}

// TestDriverAttach checks bearer attach by provider id and failure that never leaks the key.
func TestDriverAttach(t *testing.T) {
	// attachCase defines the key and session used for one attach.
	type attachCase struct {
		name    string
		key     string
		session service.InterviewVoiceSession
		wantErr error
	}
	noProvider := session()
	noProvider.ProviderSessionID = ""
	tests := []attachCase{
		{name: "attaches to the provider session", key: testKey, session: session()},
		{name: "empty key is refused", key: "", session: session(), wantErr: voicesession.ErrSessionUnavailable},
		{name: "missing provider id is refused", key: testKey, session: noProvider, wantErr: interviewvoice.ErrAttachFailed},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := newProvider(t, http.StatusCreated, "")
			d := p.driver(tt.key)
			err := d.Attach(context.Background(), tt.session)
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				assert.NotContains(t, err.Error(), testKey)
				return
			}
			require.NoError(t, err)
			t.Cleanup(func() { _ = d.Close(context.Background(), tt.session) })
			require.NoError(t, d.Control(context.Background(), service.InterviewVoiceControl{Session: tt.session, Command: "ping"}))
			<-p.received
			key, id := p.attached()
			assert.Equal(t, "Bearer "+testKey, key)
			assert.Equal(t, providerID, id)
		})
	}
}

// TestDriverSend checks that context and control reach the sideband and require an attach.
func TestDriverSend(t *testing.T) {
	// sendCase selects one outbound operation.
	type sendCase struct {
		name   string
		attach bool
		send   func(*interviewvoice.Driver) error
		want   map[string]any
		err    error
	}
	tests := []sendCase{
		{
			name: "supply context carries the text", attach: true,
			send: func(d *interviewvoice.Driver) error {
				return d.SupplyContext(context.Background(), service.InterviewVoiceContext{Session: session(), Text: "stage background"})
			},
			want: map[string]any{"instructions": "stage background"},
		},
		{
			name: "control carries command and payload", attach: true,
			send: func(d *interviewvoice.Driver) error {
				return d.Control(context.Background(), service.InterviewVoiceControl{Session: session(), Command: "response.cancel", Payload: []byte(`{"reason":"interrupt"}`)})
			},
			want: map[string]any{"type": "response.cancel", "reason": "interrupt"},
		},
		{
			name: "supply context needs an attach",
			send: func(d *interviewvoice.Driver) error {
				return d.SupplyContext(context.Background(), service.InterviewVoiceContext{Session: session(), Text: "x"})
			},
			err: interviewvoice.ErrNotAttached,
		},
		{
			name: "control needs an attach",
			send: func(d *interviewvoice.Driver) error {
				return d.Control(context.Background(), service.InterviewVoiceControl{Session: session(), Command: "ping"})
			},
			err: interviewvoice.ErrNotAttached,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := newProvider(t, http.StatusCreated, "")
			d := p.driver(testKey)
			if tt.attach {
				require.NoError(t, d.Attach(context.Background(), session()))
				t.Cleanup(func() { _ = d.Close(context.Background(), session()) })
			}
			err := tt.send(d)
			if tt.err != nil {
				require.ErrorIs(t, err, tt.err)
				return
			}
			require.NoError(t, err)
			var frame map[string]any
			require.NoError(t, json.Unmarshal([]byte(<-p.received), &frame))
			for key, want := range tt.want {
				if key == "instructions" {
					sess, _ := frame["session"].(map[string]any)
					assert.Equal(t, want, sess[key])
					continue
				}
				assert.Equal(t, want, frame[key])
			}
		})
	}
}

// TestDriverClose checks best-effort provider close, returned failure, and released ownership.
func TestDriverClose(t *testing.T) {
	// closeCase decides whether the caller's context is already ended.
	type closeCase struct {
		name      string
		cancelled bool
		wantErr   error
	}
	tests := []closeCase{
		{name: "sends session close and releases"},
		{name: "failure is returned yet resources are released", cancelled: true, wantErr: context.Canceled},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := newProvider(t, http.StatusCreated, "")
			d := p.driver(testKey)
			require.NoError(t, d.Attach(context.Background(), session()))
			ctx := context.Background()
			if tt.cancelled {
				var stop context.CancelFunc
				ctx, stop = context.WithCancel(ctx)
				stop()
			}
			err := d.Close(ctx, session())
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
				assert.JSONEq(t, `{"type":"session.close"}`, <-p.received)
			}
			err = d.Control(context.Background(), service.InterviewVoiceControl{Session: session(), Command: "ping"})
			assert.ErrorIs(t, err, interviewvoice.ErrNotAttached)
		})
	}
}

// TestDriverCloseReattaches proves a close after a restart attaches with the stored provider id and sends session.close.
func TestDriverCloseReattaches(t *testing.T) {
	// reattachCase names the session identity handed to a driver that holds no connection for it.
	type reattachCase struct {
		name       string
		providerID string
		wantAttach bool
	}
	tests := []reattachCase{
		{name: "stored provider id attaches then closes", providerID: providerID, wantAttach: true},
		{name: "no provider id has nothing to close"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := newProvider(t, http.StatusCreated, "")
			d := p.driver(testKey)
			s := session()
			s.ProviderSessionID = tt.providerID

			err := d.Close(context.Background(), s)

			require.NoError(t, err)
			if !tt.wantAttach {
				_, id := p.attached()
				assert.Empty(t, id)
				return
			}
			select {
			case frame := <-p.received:
				assert.JSONEq(t, `{"type":"session.close"}`, frame)
			case <-time.After(2 * time.Second):
				t.Fatal("no session.close reached the provider")
			}
			key, id := p.attached()
			assert.Equal(t, providerID, id)
			assert.Equal(t, "Bearer "+testKey, key)
		})
	}
}
