package candidateinterviewhandler

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/rfanazhari/amer-prototype/translations"
)

const maxBodyBytes = 1 << 20

var errInput = errors.New("invalid input")

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

// formatted renders a UTC instant with exactly three fractional digits.
func formatted(v time.Time) string {
	return v.UTC().Format("2006-01-02T15:04:05.000Z")
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

// readJSON accepts one bounded JSON object and rejects unknown fields and trailing documents.
func readJSON(r *http.Request, dst any) error {
	body, err := io.ReadAll(io.LimitReader(r.Body, maxBodyBytes+1))
	if err != nil || len(body) > maxBodyBytes {
		return errInput
	}
	trimmed := bytes.TrimSpace(body)
	if len(trimmed) == 0 || trimmed[0] != '{' {
		return errInput
	}
	members := json.NewDecoder(bytes.NewReader(body))
	if _, err := members.Token(); err != nil {
		return errInput
	}
	seen := map[string]bool{}
	for members.More() {
		key, err := members.Token()
		if err != nil {
			return errInput
		}
		name, ok := key.(string)
		if !ok || seen[name] {
			return errInput
		}
		seen[name] = true
		var value json.RawMessage
		if members.Decode(&value) != nil {
			return errInput
		}
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

// requireOrigin rejects requests whose Origin is not the configured access origin.
// Same-origin GET and HEAD omit Origin in browsers; those are accepted when Host matches that origin.
func requireOrigin(r *http.Request, origin string) error {
	if origin == "" {
		return errInput
	}
	got := r.Header.Get("Origin")
	if got == origin {
		return nil
	}
	if got == "" && (r.Method == http.MethodGet || r.Method == http.MethodHead) && r.Host == originHost(origin) {
		return nil
	}
	return errInput
}

// originHost returns the host:port of a configured absolute origin.
func originHost(origin string) string {
	parsed, err := url.Parse(origin)
	if err != nil {
		return ""
	}
	return parsed.Host
}

// requireJSONContentType rejects bodies that are not application/json.
func requireJSONContentType(r *http.Request) error {
	ct := r.Header.Get("Content-Type")
	if ct != "application/json" && !bytes.HasPrefix([]byte(ct), []byte("application/json;")) {
		return errInput
	}
	return nil
}
