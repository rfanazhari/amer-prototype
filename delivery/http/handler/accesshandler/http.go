package accesshandler

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
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

// loginRequest accepts credential fields only; system-owned members are rejected as unknown.
type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// provisionRequest accepts HR identity fields chosen by the current ADMIN.
type provisionRequest struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

// emptyRequest permits an absent or empty logout object without unexpected fields.
type emptyRequest struct{}

// candidateRequest accepts candidate contact fields chosen by current HR.
type candidateRequest struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

// exchangeRequest accepts the invitation secret only; assignment overrides are unknown fields.
type exchangeRequest struct {
	Token string `json:"token"`
}

// candidateBody is the candidate identity created or read by HR without credentials.
type candidateBody struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Email     string `json:"email"`
	CreatedAt string `json:"createdAt"`
	UpdatedAt string `json:"updatedAt"`
}

// invitationBody is the issued invitation including the plaintext token shown once.
type invitationBody struct {
	ID           string `json:"id"`
	AssignmentID string `json:"assignmentId"`
	ExpiresAt    string `json:"expiresAt"`
	Token        string `json:"token"`
}

// candidateIdentityBody is the safe candidate identity returned after exchange or session reads.
type candidateIdentityBody struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// assignmentRefBody is the assignment identity bound to a candidate session.
type assignmentRefBody struct {
	ID string `json:"id"`
}

// candidateAccessBody is the exchange or current-session payload without cookie material.
type candidateAccessBody struct {
	Candidate  candidateIdentityBody `json:"candidate"`
	Assignment assignmentRefBody     `json:"assignment"`
	Session    sessionBody           `json:"session"`
	CSRFToken  string                `json:"csrfToken,omitempty"`
}

// accountBody is the safe administrative identity returned after login or session reads.
type accountBody struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Email  string `json:"email"`
	Role   string `json:"role"`
	Status string `json:"status,omitempty"`
}

// sessionBody is session metadata; the cookie remains the credential.
type sessionBody struct {
	ID        string `json:"id"`
	ExpiresAt string `json:"expiresAt"`
}

// loginBody is the login success payload without cookie material.
type loginBody struct {
	Account accountBody `json:"account"`
	Session sessionBody `json:"session"`
}

// currentBody is the current-session payload including CSRF material.
type currentBody struct {
	Account   accountBody `json:"account"`
	Session   sessionBody `json:"session"`
	CSRFToken string      `json:"csrfToken"`
}

// provisionBody is the HR identity created by ADMIN without a password.
type provisionBody struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Email     string `json:"email"`
	Role      string `json:"role"`
	Status    string `json:"status"`
	CreatedAt string `json:"createdAt"`
	UpdatedAt string `json:"updatedAt"`
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

// readJSON accepts one bounded JSON object, rejects unknown fields and trailing documents, and optionally permits an absent body.
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
