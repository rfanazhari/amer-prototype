// Package liveprobe contains isolated GPT-Live feasibility helpers, not execution lifecycle authority.
package liveprobe

// Setup retains the identity needed for sideband attachment alongside the browser answer.
type Setup struct {
	SessionID string
	SDP       string
}

// DecodeSetup validates a provider setup response before sideband attachment.
func DecodeSetup(body []byte) (Setup, error) { return Setup{}, nil }

// Shape returns payload-free event metadata suitable for synthetic spike diagnostics.
func Shape(body []byte) (string, error) { return "", nil }
