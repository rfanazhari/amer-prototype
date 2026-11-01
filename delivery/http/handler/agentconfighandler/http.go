package agentconfighandler

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"

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

// readJSON accepts one bounded JSON object, rejects unknown fields, duplicates and trailing documents.
func readJSON(r *http.Request, dst any, emptyAllowed bool) error {
	body, err := io.ReadAll(io.LimitReader(r.Body, maxBodyBytes+1))
	if err != nil || len(body) > maxBodyBytes {
		return errInput
	}
	if len(bytes.TrimSpace(body)) == 0 && emptyAllowed {
		return nil
	}
	trimmed := bytes.TrimSpace(body)
	if len(trimmed) == 0 || trimmed[0] != '{' {
		return errInput
	}
	if err = rejectDuplicateKeys(trimmed); err != nil {
		return err
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

// rejectDuplicateKeys walks JSON tokens and rejects repeated object member names.
func rejectDuplicateKeys(raw []byte) error {
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	return walkJSON(dec)
}

// walkJSON consumes one JSON value and rejects duplicate keys in nested objects.
func walkJSON(dec *json.Decoder) error {
	tok, err := dec.Token()
	if err != nil {
		return errInput
	}
	delim, ok := tok.(json.Delim)
	if !ok {
		return nil
	}
	switch delim {
	case '{':
		seen := map[string]struct{}{}
		for dec.More() {
			keyTok, err := dec.Token()
			if err != nil {
				return errInput
			}
			key, ok := keyTok.(string)
			if !ok {
				return errInput
			}
			if _, exists := seen[key]; exists {
				return errInput
			}
			seen[key] = struct{}{}
			if err = walkJSON(dec); err != nil {
				return err
			}
		}
		if _, err = dec.Token(); err != nil {
			return errInput
		}
	case '[':
		for dec.More() {
			if err = walkJSON(dec); err != nil {
				return err
			}
		}
		if _, err = dec.Token(); err != nil {
			return errInput
		}
	default:
		return errInput
	}
	return nil
}

// requireOrigin rejects requests whose Origin is missing or not the configured access origin.
func requireOrigin(r *http.Request, origin string) error {
	if origin == "" || r.Header.Get("Origin") != origin {
		return errInput
	}
	return nil
}

// requireJSONContentType rejects bodies that are not application/json.
func requireJSONContentType(r *http.Request, hasBody bool) error {
	if !hasBody {
		return nil
	}
	ct := r.Header.Get("Content-Type")
	if ct != "application/json" && !bytes.HasPrefix([]byte(ct), []byte("application/json;")) {
		return errInput
	}
	return nil
}
