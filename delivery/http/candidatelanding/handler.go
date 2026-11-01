package candidatelanding

import (
	"encoding/json"
	"net/http"

	"github.com/rfanazhari/amer-prototype/pkg/errs"
	"github.com/rfanazhari/amer-prototype/translations"
)

// Handler serves GET /candidate/access without exchanging an invitation.
type Handler struct{}

// envelope keeps status and data present on every JSON error response.
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

// fail localizes a stable error code and emits an empty data object in the error envelope.
func fail(w http.ResponseWriter, r *http.Request, status int, code string) {
	msg := translations.Message(r.Header.Get("Accept-Language"), code)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(envelope{Status: "error", Data: map[string]any{}, Error: &errorBody{Code: code, Message: msg}})
}

// ServeHTTP currently rejects landing until the invitation page is implemented.
func (Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	fail(w, r, http.StatusNotFound, errs.NotFound)
}
