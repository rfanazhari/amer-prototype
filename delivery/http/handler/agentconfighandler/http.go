package agentconfighandler

import (
	"encoding/json"
	"net/http"

	"github.com/rfanazhari/amer-prototype/translations"
)

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

// respond writes the standard JSON envelope, no-store cache policy and selected HTTP status.
func respond(w http.ResponseWriter, status int, body envelope) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

// fail localizes a stable error code and emits an empty data object in the error envelope.
func fail(w http.ResponseWriter, r *http.Request, status int, code string) {
	msg := translations.Message(r.Header.Get("Accept-Language"), code)
	respond(w, status, envelope{Status: "error", Data: map[string]any{}, Error: &errorBody{Code: code, Message: msg}})
}
