// Package interviewvoice adapts GPT-Live transport to the interview driver port.
package interviewvoice

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/SALT-Indonesia/salt-pkg/clientmanager"
	"github.com/SALT-Indonesia/salt-pkg/logmanager"
	"github.com/rfanazhari/amer-prototype/application/service"
	"github.com/rfanazhari/amer-prototype/application/usecase/voicesession"
	"golang.org/x/net/websocket"
)

const (
	defaultSessionsURL = "https://api.openai.com/v1/live/sessions"
	defaultAttachURL   = "wss://api.openai.com/v1/live/sessions"
	attachOrigin       = "https://api.openai.com"
	defaultModel       = "gpt-live-1"
	defaultVoice       = "marin"
	defaultTimeout     = 30 * time.Second
	maxSetupBytes      = 1 << 20
)

var (
	// ErrNotAttached reports a sideband operation for a session without an attached connection.
	ErrNotAttached = errors.New("interview voice session is not attached")
	// ErrAttachFailed reports that backend control could not be opened for an existing provider session.
	ErrAttachFailed = errors.New("interview voice attach failed")
	// ErrTransport reports a failed send, receive or close on an attached sideband.
	ErrTransport = errors.New("interview voice transport failed")

	providerID = regexp.MustCompile(`^[A-Za-z0-9_-]{1,200}$`)
)

// Driver talks to GPT-Live with server-owned credentials. Zero values select provider defaults.
// It retains attached connections until Close, a disconnect, or its owner's context ends.
type Driver struct {
	APIKey string
	Model  string
	Voice  string
	// SessionsURL is the Live sessions endpoint; AttachURL is the sessions base for sideband attach.
	SessionsURL string
	AttachURL   string
	// Timeout bounds the setup request; zero means 30 seconds.
	Timeout time.Duration

	mu    sync.Mutex
	conns map[string]*websocket.Conn
}

// liveSetup is the JSON body accepted by POST /v1/live/sessions.
type liveSetup struct {
	Session   liveSession   `json:"session"`
	Transport liveTransport `json:"transport"`
}

// liveSession is the startup configuration sent with a Live WebRTC offer.
type liveSession struct {
	Model        string    `json:"model"`
	Instructions string    `json:"instructions"`
	Audio        liveAudio `json:"audio"`
}

// liveAudio holds the output voice for a Live session.
type liveAudio struct {
	Output liveAudioOutput `json:"output"`
}

// liveAudioOutput names the voice used for Live speech.
type liveAudioOutput struct {
	Voice string `json:"voice"`
}

// liveTransport carries an SDP body and its kind.
type liveTransport struct {
	SDP  string `json:"sdp"`
	Type string `json:"type"`
}

// liveAnswer is the provider JSON that carries the session identity and SDP answer.
type liveAnswer struct {
	Session struct {
		ID string `json:"id"`
	} `json:"session"`
	Transport liveTransport `json:"transport"`
}

// normalizeSDP turns every line ending into CRLF and keeps a terminator on the last line,
// matching the voice-session prototype rule.
func normalizeSDP(offer string) string {
	sdp := strings.ReplaceAll(strings.ReplaceAll(offer, "\r\n", "\n"), "\n", "\r\n")
	if !strings.HasSuffix(sdp, "\r\n") {
		sdp += "\r\n"
	}
	return sdp
}

// orDefault returns configured unless it is empty.
func orDefault(configured, fallback string) string {
	if configured == "" {
		return fallback
	}
	return configured
}

// Create opens one provider session and retains its id with the SDP answer.
// Every failure maps to the setup sentinel without provider text or credentials.
func (d *Driver) Create(ctx context.Context, req service.InterviewVoiceCreate) (service.InterviewVoiceCreated, error) {
	if d.APIKey == "" {
		logmanager.InfoWithContext(ctx, "interview voice setup skipped", map[string]string{"cause": "provider key is empty"})
		return service.InterviewVoiceCreated{}, voicesession.ErrSessionUnavailable
	}
	timeout := d.Timeout
	if timeout <= 0 {
		timeout = defaultTimeout
	}
	payload := liveSetup{
		Session: liveSession{
			Model:        orDefault(d.Model, defaultModel),
			Instructions: req.Instructions,
			Audio:        liveAudio{Output: liveAudioOutput{Voice: orDefault(d.Voice, defaultVoice)}},
		},
		Transport: liveTransport{SDP: normalizeSDP(req.SDP), Type: "webrtc"},
	}
	response, err := clientmanager.CallBytes(
		ctx,
		orDefault(d.SessionsURL, defaultSessionsURL),
		clientmanager.WithMethod(http.MethodPost),
		clientmanager.WithAuth(clientmanager.AuthBearer(d.APIKey)),
		clientmanager.WithRequestBody(payload),
		clientmanager.WithTimeout(timeout),
		clientmanager.WithMaxResponseBytes(maxSetupBytes),
	)
	if err != nil || response == nil || !response.IsSuccess() {
		return service.InterviewVoiceCreated{}, voicesession.ErrSessionUnavailable
	}
	var answer liveAnswer
	if json.Unmarshal(response.Body, &answer) != nil || !providerID.MatchString(answer.Session.ID) || answer.Transport.SDP == "" {
		return service.InterviewVoiceCreated{}, voicesession.ErrSessionUnavailable
	}
	created := req.Session
	created.ProviderSessionID = answer.Session.ID
	return service.InterviewVoiceCreated{Session: created, SDP: answer.Transport.SDP}, nil
}

var _ service.InterviewDriver = (*Driver)(nil)
