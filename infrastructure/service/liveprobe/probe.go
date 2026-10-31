// Package liveprobe contains isolated GPT-Live feasibility helpers, not execution lifecycle authority.
package liveprobe

import (
	"encoding/json"
	"errors"
	"regexp"
)

var identifier = regexp.MustCompile(`^[A-Za-z0-9_-]{1,200}$`)
var eventType = regexp.MustCompile(`^[a-z][a-z0-9_.]{0,160}$`)

// sessionIdentity represents the provider identity without retaining other session settings.
type sessionIdentity struct {
	ID string `json:"id"`
}

// transportAnswer contains only negotiated WebRTC transport data.
type transportAnswer struct {
	SDP  string `json:"sdp"`
	Type string `json:"type"`
}

// setupResponse is the minimal GPT-Live response needed by this spike.
type setupResponse struct {
	Session   sessionIdentity `json:"session"`
	Transport transportAnswer `json:"transport"`
}

// Setup retains the identity needed for sideband attachment alongside the browser answer.
type Setup struct {
	SessionID string
	SDP       string
}

// DecodeSetup validates a provider setup response before sideband attachment.
func DecodeSetup(body []byte) (Setup, error) {
	var response setupResponse
	if json.Unmarshal(body, &response) != nil || !identifier.MatchString(response.Session.ID) || response.Transport.Type != "webrtc" || response.Transport.SDP == "" {
		return Setup{}, errors.New("invalid live setup response")
	}
	return Setup{SessionID: response.Session.ID, SDP: response.Transport.SDP}, nil
}

// Shape returns payload-free event metadata suitable for synthetic spike diagnostics.
func Shape(body []byte) (string, error) {
	var fields map[string]json.RawMessage
	if json.Unmarshal(body, &fields) != nil {
		return "", errors.New("invalid event")
	}
	var kind string
	if json.Unmarshal(fields["type"], &kind) != nil || !eventType.MatchString(kind) {
		return "", errors.New("invalid event type")
	}
	return kind, nil
}
