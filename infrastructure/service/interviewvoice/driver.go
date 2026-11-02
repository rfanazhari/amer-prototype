// Package interviewvoice adapts GPT-Live transport to the interview driver port.
package interviewvoice

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/rfanazhari/amer-prototype/application/service"
	"golang.org/x/net/websocket"
)

var (
	// ErrNotAttached reports a sideband operation for a session without an attached connection.
	ErrNotAttached = errors.New("interview voice session is not attached")
	// ErrAttachFailed reports that backend control could not be opened for an existing provider session.
	ErrAttachFailed = errors.New("interview voice attach failed")
	// ErrTransport reports a failed send, receive or close on an attached sideband.
	ErrTransport = errors.New("interview voice transport failed")
	// errNotImplemented marks behavior that the transport does not provide yet.
	errNotImplemented = errors.New("interview voice transport is not implemented")
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

// Create is not implemented yet.
func (d *Driver) Create(context.Context, service.InterviewVoiceCreate) (service.InterviewVoiceCreated, error) {
	return service.InterviewVoiceCreated{}, errNotImplemented
}

// Attach is not implemented yet.
func (d *Driver) Attach(context.Context, service.InterviewVoiceSession) error {
	return errNotImplemented
}

// Observe is not implemented yet.
func (d *Driver) Observe(context.Context, service.InterviewVoiceSession) (service.InterviewVoiceObservation, error) {
	return service.InterviewVoiceObservation{}, errNotImplemented
}

// SupplyContext is not implemented yet.
func (d *Driver) SupplyContext(context.Context, service.InterviewVoiceContext) error {
	return errNotImplemented
}

// Control is not implemented yet.
func (d *Driver) Control(context.Context, service.InterviewVoiceControl) error {
	return errNotImplemented
}

// Close is not implemented yet.
func (d *Driver) Close(context.Context, service.InterviewVoiceSession) error {
	return errNotImplemented
}

var _ service.InterviewDriver = (*Driver)(nil)
