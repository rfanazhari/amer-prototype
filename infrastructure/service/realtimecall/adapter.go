// Package realtimecall sends one Realtime calls request for voice-session setup.
package realtimecall

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"

	"github.com/rfanazhari/amer-prototype/application/usecase/voicesession"
)

// Adapter posts one Realtime calls request using server-owned credentials and session settings.
type Adapter struct {
	APIKey string
	Model  string
	Voice  string
	URL    string
}

// sessionPayload is the Realtime session JSON attached beside the SDP offer.
type sessionPayload struct {
	Type         string       `json:"type"`
	Model        string       `json:"model"`
	Voice        string       `json:"voice"`
	Instructions string       `json:"instructions"`
	Audio        sessionAudio `json:"audio"`
}

// sessionAudio holds the audio subtree of a Realtime session payload.
type sessionAudio struct {
	Input sessionAudioInput `json:"input"`
}

// sessionAudioInput holds input audio settings, including turn detection.
type sessionAudioInput struct {
	TurnDetection sessionTurnDetection `json:"turn_detection"`
}

// sessionTurnDetection configures provider-owned end-of-turn replies.
type sessionTurnDetection struct {
	Type              string `json:"type"`
	CreateResponse    bool   `json:"create_response"`
	InterruptResponse bool   `json:"interrupt_response"`
}

// unavailable maps every adapter failure to the setup sentinel without extra cause text.
func unavailable() (string, error) {
	return "", voicesession.ErrSessionUnavailable
}

// writeMultipart builds the Realtime calls body with sdp and session fields.
func writeMultipart(offer string, session []byte) (*bytes.Buffer, string, error) {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	if err := writer.WriteField("sdp", offer); err != nil {
		return nil, "", err
	}
	if err := writer.WriteField("session", string(session)); err != nil {
		return nil, "", err
	}
	if err := writer.Close(); err != nil {
		return nil, "", err
	}
	return body, writer.FormDataContentType(), nil
}

// Exchange sends one authorized multipart Realtime calls request and returns the SDP answer body.
func (a Adapter) Exchange(ctx context.Context, offer string, instructions string) (string, error) {
	if a.APIKey == "" {
		return unavailable()
	}
	session, err := json.Marshal(sessionPayload{
		Type:         "realtime",
		Model:        a.Model,
		Voice:        a.Voice,
		Instructions: instructions,
		Audio: sessionAudio{Input: sessionAudioInput{TurnDetection: sessionTurnDetection{
			Type:              "server_vad",
			CreateResponse:    true,
			InterruptResponse: true,
		}}},
	})
	if err != nil {
		return unavailable()
	}
	body, contentType, err := writeMultipart(offer, session)
	if err != nil {
		return unavailable()
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, a.URL, body)
	if err != nil {
		return unavailable()
	}
	request.Header.Set("Authorization", "Bearer "+a.APIKey)
	request.Header.Set("Content-Type", contentType)
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return unavailable()
	}
	defer func() { _ = response.Body.Close() }()
	payload, err := io.ReadAll(response.Body)
	if err != nil {
		return unavailable()
	}
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return unavailable()
	}
	return string(payload), nil
}

var _ voicesession.Negotiator = Adapter{}
