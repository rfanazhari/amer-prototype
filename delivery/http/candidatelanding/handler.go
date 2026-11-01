package candidatelanding

import (
	_ "embed"
	"encoding/json"
	"net/http"

	"github.com/rfanazhari/amer-prototype/pkg/errs"
	"github.com/rfanazhari/amer-prototype/translations"
)

// page is the embedded invitation landing document for GET /candidate/access.
//
//go:embed page.html
var page []byte

// Handler serves GET /candidate/access without exchanging an invitation on GET.
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

// ServeHTTP writes the embedded landing page for GET and rejects other methods without exchanging.
func (Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		fail(w, r, http.StatusMethodNotAllowed, errs.MethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Referrer-Policy", "no-referrer")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(page)
}
