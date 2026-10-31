// Package realtimecall sends one Realtime calls request for voice-session setup.
package realtimecall

import (
	"context"

	"github.com/rfanazhari/amer-prototype/application/usecase/voicesession"
)

// Adapter posts one Realtime calls request using server-owned credentials and session settings.
type Adapter struct {
	APIKey string
	Model  string
	Voice  string
	URL    string
}

// Exchange is an inert RED stub until the outbound Realtime call is implemented.
func (Adapter) Exchange(context.Context, string, string) (string, error) {
	return "", nil
}

var _ voicesession.Negotiator = Adapter{}
