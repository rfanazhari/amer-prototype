// Package realtimecall sends one Realtime calls request for voice-session setup.
package realtimecall

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/SALT-Indonesia/salt-pkg/clientmanager"
	"github.com/SALT-Indonesia/salt-pkg/logmanager"
	"github.com/rfanazhari/amer-prototype/application/usecase/voicesession"
)

const (
	realtimeCallsURL = "https://api.openai.com/v1/realtime/calls"
	liveSessionsURL  = "https://api.openai.com/v1/live/sessions"
	realtimeModel    = "gpt-realtime-2.1"
	liveModel        = "gpt-live-1"
)

// Adapter posts one Realtime calls request using server-owned credentials and session settings.
type Adapter struct {
	APIKey string
	Model  string
	Voice  string
	URL    string
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

// liveTransport carries the browser SDP offer.
type liveTransport struct {
	SDP  string `json:"sdp"`
	Type string `json:"type"`
}

// liveAnswer is the provider JSON that contains the SDP answer.
type liveAnswer struct {
	Transport liveTransport `json:"transport"`
}

// unavailable maps every adapter failure to the setup sentinel without extra cause text.
func unavailable() (string, error) {
	return "", voicesession.ErrSessionUnavailable
}

// note records a setup failure that happens before the provider call.
func note(ctx context.Context, message, cause string) {
	logmanager.InfoWithContext(ctx, message, map[string]string{"cause": cause})
}

// normalizeOffer gives the provider parser a terminated SDP block. An offer that ends mid-line is unmarshaled as EOF.
func normalizeOffer(offer string) string {
	if strings.HasSuffix(offer, "\n") {
		return offer
	}
	return offer + "\r\n"
}

// setupEndpoint returns the Live sessions URL for the rejected realtime calls endpoint.
func setupEndpoint(configured string) string {
	if configured == "" || configured == realtimeCallsURL {
		return liveSessionsURL
	}
	return configured
}

// setupModel returns the Live model when the configured value is the realtime calls default.
func setupModel(configured string) string {
	if configured == "" || configured == realtimeModel {
		return liveModel
	}
	return configured
}

// Exchange sends one authorized Live session request and returns the SDP answer.
func (a Adapter) Exchange(ctx context.Context, offer string, instructions string) (string, error) {
	if a.APIKey == "" {
		note(ctx, "realtime setup skipped", "provider key is empty")
		return unavailable()
	}
	endpoint := setupEndpoint(a.URL)
	model := setupModel(a.Model)
	voice := a.Voice
	if voice == "" {
		voice = "marin"
	}
	normalized := normalizeOffer(offer)
	payload := liveSetup{
		Session: liveSession{
			Model:        model,
			Instructions: instructions,
			Audio:        liveAudio{Output: liveAudioOutput{Voice: voice}},
		},
		Transport: liveTransport{SDP: normalized, Type: "webrtc"},
	}
	response, err := clientmanager.CallBytes(
		ctx,
		endpoint,
		clientmanager.WithMethod(http.MethodPost),
		clientmanager.WithAuth(clientmanager.AuthBearer(a.APIKey)),
		clientmanager.WithRequestBody(payload),
	)
	if err != nil || response == nil || !response.IsSuccess() {
		return unavailable()
	}
	var answer liveAnswer
	if err = json.Unmarshal(response.Body, &answer); err != nil || answer.Transport.SDP == "" {
		return unavailable()
	}
	return answer.Transport.SDP, nil
}

var _ voicesession.Negotiator = Adapter{}
