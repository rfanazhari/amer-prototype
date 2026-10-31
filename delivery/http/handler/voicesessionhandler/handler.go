// Package voicesessionhandler maps voice-session setup HTTP requests to the JSON envelope.
package voicesessionhandler

import (
	"net/http"

	"github.com/rfanazhari/amer-prototype/application/usecase/voicesession"
)

// Handler translates a voice-session setup request through NegotiateOperation.
type Handler struct {
	Negotiate voicesession.NegotiateOperation
	LogError  func(error)
}

// ServeHTTP accepts POST setup requests and writes the JSON envelope.
func (h Handler) ServeHTTP(http.ResponseWriter, *http.Request) {}
