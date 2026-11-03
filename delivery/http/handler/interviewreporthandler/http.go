package interviewreporthandler

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/rfanazhari/amer-prototype/application/repository/interviewreportport"
	"github.com/rfanazhari/amer-prototype/application/service"
	"github.com/rfanazhari/amer-prototype/domain/aggregate/interviewreport"
	"github.com/rfanazhari/amer-prototype/pkg/errs"
	"github.com/rfanazhari/amer-prototype/translations"
)

// envelope preserves the shared success and failure response shape.
type envelope struct {
	Status string     `json:"status"`
	Data   any        `json:"data"`
	Error  *errorBody `json:"error,omitempty"`
}

// errorBody exposes only a stable code and a localized safe message.
type errorBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// write emits JSON with confidential report responses excluded from caches.
func write(w http.ResponseWriter, status int, body envelope) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

// respond wraps a successful transport view in the standard envelope.
func respond(w http.ResponseWriter, status int, data any) {
	write(w, status, envelope{Status: "success", Data: data})
}

// fail localizes a public code without serializing diagnostic causes.
func fail(w http.ResponseWriter, r *http.Request, status int, code string) {
	write(w, status, envelope{Status: "error", Data: map[string]any{}, Error: &errorBody{Code: code, Message: translations.Message(r.Header.Get("Accept-Language"), code)}})
}

// sendError preserves report error categories and gives technical wrappers precedence over their causes.
func (h Handler) sendError(w http.ResponseWriter, r *http.Request, err error) {
	var technical errs.Technical
	if errors.As(err, &technical) {
		h.internal(w, r, err)
		return
	}
	switch {
	case errors.Is(err, service.ErrSessionUnbound), errors.Is(err, service.ErrCandidateSessionUnbound), errors.Is(err, service.ErrAuthenticationFailed):
		fail(w, r, 401, errs.IRPUnauthenticated)
	case errors.Is(err, service.ErrForbidden), errors.Is(err, service.ErrCSRFMismatch):
		fail(w, r, 403, errs.IRPForbidden)
	case errors.Is(err, service.ErrReportInvalidInput), errors.Is(err, service.ErrInvalidInput):
		fail(w, r, 400, errs.InvalidInput)
	case errors.Is(err, interviewreportport.ErrNotFound):
		fail(w, r, 404, errs.NotFound)
	case errors.Is(err, service.ErrReportIdempotencyConflict):
		fail(w, r, 409, errs.IRPIdempotencyConflict)
	case errors.Is(err, service.ErrReportNotCompleted):
		fail(w, r, 409, errs.IRPNotCompleted)
	case errors.Is(err, service.ErrReportEvidenceUnavailable):
		fail(w, r, 409, errs.IRPEvidenceUnavailable)
	case errors.Is(err, interviewreport.ErrGenerationActive):
		fail(w, r, 409, errs.IRPGenerationActive)
	case errors.Is(err, interviewreportport.ErrDependencyUnavailable):
		fail(w, r, 503, errs.IRPDependencyUnavailable)
	default:
		h.internal(w, r, err)
	}
}

// internal retains diagnostic logging outside the safe public error envelope.
func (h Handler) internal(w http.ResponseWriter, r *http.Request, err error) {
	if h.LogError != nil {
		h.LogError(err)
	}
	fail(w, r, http.StatusInternalServerError, errs.InternalError)
}
