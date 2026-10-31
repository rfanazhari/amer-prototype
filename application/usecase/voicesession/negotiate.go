// Package voicesession coordinates one-shot voice-session setup against a provider port.
package voicesession

import (
	"context"
	"errors"
	"strings"
)

const interviewerInstructions = "You are a simple interviewer in a feasibility prototype. Speak Indonesian. English technical terms from the candidate are allowed and should be understood. When the first response is requested, greet the candidate in Indonesian and ask one opening question before waiting for an answer. On each later turn, stay in this same conversation, use what the candidate already said, and refer to an earlier answer from this session. Do not score, grade, pass, or fail any answer. Do not select questions from a fixed HR list, and do not apply compensation or hiring policy."

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
func (u Negotiate) Execute(ctx context.Context, offer string) (string, error) {
	trimmed := strings.TrimSpace(offer)
	if trimmed == "" {
		return "", ErrInvalidOffer
	}
	answer, err := u.Negotiator.Exchange(ctx, trimmed, interviewerInstructions)
	if err != nil {
		return "", ErrSessionUnavailable
	}
	return answer, nil
}
