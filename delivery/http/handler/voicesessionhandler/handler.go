// Package voicesessionhandler maps voice-session setup HTTP requests to the JSON envelope.
package voicesessionhandler

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/rfanazhari/amer-prototype/application/usecase/voicesession"
	"github.com/rfanazhari/amer-prototype/pkg/errs"
	"github.com/rfanazhari/amer-prototype/translations"
)

// maxSetupBodyBytes is the 1 MiB JSON body limit shared with Task requests.
const maxSetupBodyBytes = 1 << 20

// errInput marks a setup body that fails transport validation before negotiation.
var errInput = errors.New("invalid input")

// Handler translates a voice-session setup request through NegotiateOperation.
type Handler struct {
	Negotiate voicesession.NegotiateOperation
	LogError  func(error)
}

// envelope keeps status and data present on every response, with error included only on failures.
type envelope struct {
	Status string     `json:"status"`
	Data   any        `json:"data"`
	Error  *errorBody `json:"error,omitempty"`
}

// errorBody carries a stable public code and a localized message without technical causes.
type errorBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// answerBody is the success data object. It contains the provider SDP answer only.
type answerBody struct {
	SDP string `json:"sdp"`
}

// setupRequest accepts the client SDP offer. Unknown fields are rejected.
type setupRequest struct {
	SDP string `json:"sdp"`
}

// respond writes the standard JSON envelope and selected HTTP status.
func respond(w http.ResponseWriter, status int, body envelope) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

// fail localizes a stable error code and emits an empty data object in the error envelope.
func fail(w http.ResponseWriter, r *http.Request, status int, code string) {
	msg := translations.Message(r.Header.Get("Accept-Language"), code)
	respond(w, status, envelope{Status: "error", Data: map[string]any{}, Error: &errorBody{Code: code, Message: msg}})
}

// readJSON accepts one bounded JSON object and rejects unknown fields, non-objects, and trailing documents.
func readJSON(r *http.Request, dst any) error {
	body, err := io.ReadAll(io.LimitReader(r.Body, maxSetupBodyBytes+1))
	if err != nil || len(body) > maxSetupBodyBytes {
		return errInput
	}
	trimmed := bytes.TrimSpace(body)
	if len(trimmed) == 0 || trimmed[0] != '{' {
		return errInput
	}
	dec := json.NewDecoder(bytes.NewReader(body))
	dec.DisallowUnknownFields()
	if err = dec.Decode(dst); err != nil {
		return errInput
	}
	var extra any
	if dec.Decode(&extra) != io.EOF {
		return errInput
	}
	return nil
}

// sendError maps classified setup failures to safe HTTP errors. Technical wrappers are logged and returned as internal errors.
func (h Handler) sendError(w http.ResponseWriter, r *http.Request, err error) {
	var technical errs.Technical
	if errors.As(err, &technical) {
		if h.LogError != nil {
			h.LogError(err)
		}
		fail(w, r, http.StatusInternalServerError, errs.InternalError)
		return
	}
	switch {
	case errors.Is(err, voicesession.ErrInvalidOffer):
		fail(w, r, http.StatusBadRequest, errs.InvalidInput)
	case errors.Is(err, voicesession.ErrSessionUnavailable):
		fail(w, r, http.StatusServiceUnavailable, errs.SessionUnavailable)
	default:
		if h.LogError != nil {
			h.LogError(err)
		}
		fail(w, r, http.StatusInternalServerError, errs.InternalError)
	}
}

// ServeHTTP accepts POST setup requests and writes the JSON envelope.
func (h Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		fail(w, r, http.StatusMethodNotAllowed, errs.MethodNotAllowed)
		return
	}
	var req setupRequest
	if err := readJSON(r, &req); err != nil {
		fail(w, r, http.StatusBadRequest, errs.InvalidInput)
		return
	}
	answer, err := h.Negotiate.Execute(r.Context(), req.SDP)
	if err != nil {
		h.sendError(w, r, err)
		return
	}
	respond(w, http.StatusCreated, envelope{Status: "success", Data: answerBody{SDP: answer}})
}
