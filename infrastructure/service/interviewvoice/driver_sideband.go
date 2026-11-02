package interviewvoice

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/rfanazhari/amer-prototype/application/service"
	"github.com/rfanazhari/amer-prototype/application/usecase/voicesession"
	"golang.org/x/net/websocket"
)

const (
	sidebandMaxPayload = 2 << 20
	closeWriteTimeout  = time.Second
	kindEvent          = "event"
	kindDisconnected   = "disconnected"
)

// conn returns the attached connection for a local session.
func (d *Driver) conn(session service.InterviewVoiceSession) (*websocket.Conn, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	c, ok := d.conns[session.SessionID]
	if !ok {
		return nil, ErrNotAttached
	}
	return c, nil
}

// release forgets and closes the connection for a local session.
func (d *Driver) release(session service.InterviewVoiceSession) {
	d.mu.Lock()
	c := d.conns[session.SessionID]
	delete(d.conns, session.SessionID)
	d.mu.Unlock()
	if c != nil {
		_ = c.Close()
	}
}

// watch ends a blocking socket operation when ctx ends; the returned stop must be called.
func watch(ctx context.Context, c *websocket.Conn) (stop func()) {
	done := make(chan struct{})
	go func() {
		select {
		case <-ctx.Done():
			_ = c.SetDeadline(time.Unix(1, 0))
		case <-done:
		}
	}()
	return func() { close(done) }
}

// Attach opens backend control for an existing provider session and retains the connection.
func (d *Driver) Attach(ctx context.Context, session service.InterviewVoiceSession) error {
	if d.APIKey == "" {
		return voicesession.ErrSessionUnavailable
	}
	if !providerID.MatchString(session.ProviderSessionID) {
		return ErrAttachFailed
	}
	cfg, err := websocket.NewConfig(orDefault(d.AttachURL, defaultAttachURL)+"/"+session.ProviderSessionID+"/attach", attachOrigin)
	if err != nil {
		return ErrAttachFailed
	}
	cfg.Header.Set("Authorization", "Bearer "+d.APIKey)
	c, err := cfg.DialContext(ctx)
	if err != nil {
		return ErrAttachFailed
	}
	c.MaxPayloadBytes = sidebandMaxPayload
	d.mu.Lock()
	if d.conns == nil {
		d.conns = map[string]*websocket.Conn{}
	}
	previous := d.conns[session.SessionID]
	d.conns[session.SessionID] = c
	d.mu.Unlock()
	if previous != nil {
		_ = previous.Close()
	}
	return nil
}

// eventType returns the provider "type" field, or empty when the payload has none.
// The value is opaque metadata; it carries no finality or coverage meaning.
func eventType(payload []byte) string {
	var fields struct {
		Type string `json:"type"`
	}
	if json.Unmarshal(payload, &fields) != nil {
		return ""
	}
	return fields.Type
}

// Observe waits for the next opaque event. A closed sideband yields a disconnected observation
// and releases the connection; cancellation returns the context error.
func (d *Driver) Observe(ctx context.Context, session service.InterviewVoiceSession) (service.InterviewVoiceObservation, error) {
	c, err := d.conn(session)
	if err != nil {
		return service.InterviewVoiceObservation{}, err
	}
	if err = ctx.Err(); err != nil {
		return service.InterviewVoiceObservation{}, err
	}
	stop := watch(ctx, c)
	defer stop()
	var body []byte
	if err = websocket.Message.Receive(c, &body); err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return service.InterviewVoiceObservation{}, ctxErr
		}
		d.release(session)
		return service.InterviewVoiceObservation{Session: session, Kind: kindDisconnected}, nil
	}
	return service.InterviewVoiceObservation{Session: session, Kind: kindEvent, EventType: eventType(body), Payload: body}, nil
}

// send writes one JSON frame to an attached session.
func (d *Driver) send(ctx context.Context, session service.InterviewVoiceSession, frame any) error {
	c, err := d.conn(session)
	if err != nil {
		return err
	}
	if err = ctx.Err(); err != nil {
		return err
	}
	stop := watch(ctx, c)
	defer stop()
	if err = websocket.JSON.Send(c, frame); err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return ctxErr
		}
		return fmt.Errorf("%w: send", ErrTransport)
	}
	return nil
}

// SupplyContext sends application-built instructions as a session update.
func (d *Driver) SupplyContext(ctx context.Context, req service.InterviewVoiceContext) error {
	frame := map[string]any{"type": "session.update", "session": map[string]string{"instructions": req.Text}}
	return d.send(ctx, req.Session, frame)
}

// Control sends an opaque command with optional JSON object fields from Payload.
func (d *Driver) Control(ctx context.Context, req service.InterviewVoiceControl) error {
	frame := map[string]any{}
	if len(req.Payload) > 0 {
		if err := json.Unmarshal(req.Payload, &frame); err != nil {
			return fmt.Errorf("%w: payload", ErrTransport)
		}
	}
	frame["type"] = req.Command
	return d.send(ctx, req.Session, frame)
}

// Close asks the provider to end the session, then always releases the local connection.
// A returned failure never implies that committed interview progress can be rolled back.
func (d *Driver) Close(ctx context.Context, session service.InterviewVoiceSession) error {
	c, err := d.conn(session)
	if err != nil {
		return nil
	}
	defer d.release(session)
	if err = ctx.Err(); err != nil {
		return err
	}
	_ = c.SetWriteDeadline(time.Now().Add(closeWriteTimeout))
	if err = websocket.Message.Send(c, `{"type":"session.close"}`); err != nil {
		return fmt.Errorf("%w: close", ErrTransport)
	}
	return nil
}
