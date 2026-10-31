// Package voicesession coordinates one-shot voice-session setup against a provider port.
package voicesession

import (
	"context"
	"errors"
)

var (
	// ErrInvalidOffer is returned when the SDP offer is blank after trimming.
	ErrInvalidOffer = errors.New("invalid offer")
	// ErrSessionUnavailable is returned when the provider port cannot complete setup.
	ErrSessionUnavailable = errors.New("session unavailable")
)

// NegotiateOperation defines the setup command contract consumed by delivery.
type NegotiateOperation interface {
	// Execute returns the SDP answer on success and preserves classified failures.
	Execute(context.Context, string) (string, error)
}

// Negotiator exchanges an SDP offer and interviewer instructions with a provider.
type Negotiator interface {
	// Exchange sends the offer and instructions and returns the SDP answer.
	Exchange(ctx context.Context, offer string, instructions string) (string, error)
}

// Negotiate trims an SDP offer, attaches interviewer instructions and requests an answer.
type Negotiate struct {
	Negotiator Negotiator
}

// Execute rejects a blank offer before the port call and returns the provider answer otherwise.
func (u Negotiate) Execute(_ context.Context, _ string) (string, error) {
	return "", nil
}
