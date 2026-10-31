package voicesessionhandler_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/rfanazhari/amer-prototype/application/usecase/voicesession"
	"github.com/rfanazhari/amer-prototype/delivery/http/handler/voicesessionhandler"
	"github.com/rfanazhari/amer-prototype/pkg/errs"
	"github.com/rfanazhari/amer-prototype/test/mocks"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// envelopeView is the decoded public JSON envelope asserted by setup contract tests.
type envelopeView struct {
	Status string          `json:"status"`
	Data   json.RawMessage `json:"data"`
	Error  *errorView      `json:"error"`
}

// errorView is the public error object inside a failure envelope.
type errorView struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// successCase is one accepted setup request and the SDP answer the operation returns.
type successCase struct {
	name, offer, answer string
	status              int
}

// inputCase is one rejected setup body and whether NegotiateOperation may be called.
type inputCase struct {
	name, body, offer, code string
	call                    bool
	status                  int
}

// methodCase is one unsupported method on the setup route.
type methodCase struct {
	name, method, code, message string
	status                      int
}

// mappedErrorCase is one operation failure and the safe public envelope it must produce.
type mappedErrorCase struct {
	name, locale, message, leaked string
	cause                         error
	status                        int
	code                          string
	logged                        bool
}

// objectKeys returns the field names of a JSON object.
func objectKeys(t *testing.T, raw json.RawMessage) []string {
	t.Helper()
	var fields map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(raw, &fields))
	names := make([]string, 0, len(fields))
	for name := range fields {
		names = append(names, name)
	}
	return names
}

// requireJSONStatus checks the HTTP status and the JSON content type before body assertions.
func requireJSONStatus(t *testing.T, w *httptest.ResponseRecorder, status int) {
	t.Helper()
	require.Equal(t, status, w.Code)
	require.Equal(t, "application/json", w.Header().Get("Content-Type"))
}

// readEnvelope decodes a setup response into its top-level object and envelope view.
func readEnvelope(t *testing.T, raw []byte) (map[string]json.RawMessage, envelopeView) {
	t.Helper()
	var keys map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(raw, &keys))
	var view envelopeView
	require.NoError(t, json.Unmarshal(raw, &view))
	return keys, view
}

// requireErrorEnvelope checks a failure envelope, its public code, and the absence of leaked text.
func requireErrorEnvelope(t *testing.T, raw []byte, statusName, code, message, leaked string) {
	t.Helper()
	keys, view := readEnvelope(t, raw)
	require.ElementsMatch(t, []string{"status", "data", "error"}, objectKeys(t, raw))
	require.Equal(t, statusName, view.Status)
	require.JSONEq(t, `{}`, string(keys["data"]))
	require.ElementsMatch(t, []string{"code", "message"}, objectKeys(t, keys["error"]))
	require.Equal(t, code, view.Error.Code)
	require.Equal(t, message, view.Error.Message)
	if leaked != "" {
		require.NotContains(t, raw, leaked)
	}
}

// TestVoiceSessionSetupSuccess checks that a created session returns only the SDP answer.
func TestVoiceSessionSetupSuccess(t *testing.T) {
	tests := []successCase{
		{name: "sdp answer only", offer: "offer-sdp", answer: "v=0", status: http.StatusCreated},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			operation := mocks.NewNegotiateOperation(t)
			operation.On("Execute", mock.Anything, tt.offer).Return(tt.answer, nil)
			handler := voicesessionhandler.Handler{Negotiate: operation}
			recorder := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodPost, "/voice-sessions", strings.NewReader(`{"sdp":"`+tt.offer+`"}`))
			handler.ServeHTTP(recorder, request)
			requireJSONStatus(t, recorder, tt.status)
			keys, view := readEnvelope(t, recorder.Body.Bytes())
			require.ElementsMatch(t, []string{"status", "data"}, objectKeys(t, recorder.Body.Bytes()))
			require.Equal(t, "success", view.Status)
			require.Nil(t, view.Error)
			var data map[string]string
			require.NoError(t, json.Unmarshal(keys["data"], &data))
			require.Equal(t, map[string]string{"sdp": tt.answer}, data)
			body := strings.ToLower(recorder.Body.String())
			require.NotContains(t, body, "authorization")
			require.NotContains(t, body, "api_key")
			require.NotContains(t, body, "score")
			require.NotContains(t, body, "transcript")
			require.NotContains(t, body, "sk-")
		})
	}
}

// TestVoiceSessionSetupRejectsInput checks transport rejection and blank-offer mapping.
func TestVoiceSessionSetupRejectsInput(t *testing.T) {
	tests := []inputCase{
		{name: "unknown field", body: `{"sdp":"offer-sdp","score":1}`, call: false, status: http.StatusBadRequest, code: errs.InvalidInput},
		{name: "missing sdp", body: `{}`, call: true, offer: "", status: http.StatusBadRequest, code: errs.InvalidInput},
		{name: "blank offer", body: `{"sdp":"   "}`, call: true, offer: "   ", status: http.StatusBadRequest, code: errs.InvalidInput},
		{name: "empty offer", body: `{"sdp":""}`, call: true, offer: "", status: http.StatusBadRequest, code: errs.InvalidInput},
		{name: "malformed json", body: `{`, call: false, status: http.StatusBadRequest, code: errs.InvalidInput},
		{name: "non object", body: `[]`, call: false, status: http.StatusBadRequest, code: errs.InvalidInput},
		{name: "multiple documents", body: `{"sdp":"offer-sdp"}{}`, call: false, status: http.StatusBadRequest, code: errs.InvalidInput},
		{name: "oversized body", body: `{"sdp":"` + strings.Repeat("a", 1<<20) + `"}`, call: false, status: http.StatusBadRequest, code: errs.InvalidInput},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			operation := mocks.NewNegotiateOperation(t)
			if tt.call {
				operation.On("Execute", mock.Anything, tt.offer).Return("", voicesession.ErrInvalidOffer)
			}
			handler := voicesessionhandler.Handler{Negotiate: operation}
			recorder := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodPost, "/voice-sessions", strings.NewReader(tt.body))
			handler.ServeHTTP(recorder, request)
			requireJSONStatus(t, recorder, tt.status)
			requireErrorEnvelope(t, recorder.Body.Bytes(), "error", tt.code, "invalid input", "invalid offer")
		})
	}
}

// TestVoiceSessionSetupRejectsMethod checks that a non-POST setup request is rejected before negotiation.
func TestVoiceSessionSetupRejectsMethod(t *testing.T) {
	tests := []methodCase{
		{name: "get", method: http.MethodGet, status: http.StatusMethodNotAllowed, code: errs.MethodNotAllowed, message: "method not allowed"},
		{name: "put", method: http.MethodPut, status: http.StatusMethodNotAllowed, code: errs.MethodNotAllowed, message: "method not allowed"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			operation := mocks.NewNegotiateOperation(t)
			handler := voicesessionhandler.Handler{Negotiate: operation}
			recorder := httptest.NewRecorder()
			request := httptest.NewRequest(tt.method, "/voice-sessions", nil)
			handler.ServeHTTP(recorder, request)
			requireJSONStatus(t, recorder, tt.status)
			requireErrorEnvelope(t, recorder.Body.Bytes(), "error", tt.code, tt.message, "")
		})
	}
}

// TestVoiceSessionSetupMapsOperationErrors checks safe 503 and 500 envelopes without technical causes.
func TestVoiceSessionSetupMapsOperationErrors(t *testing.T) {
	secret := "sk-live-secret"
	tests := []mappedErrorCase{
		{
			name: "session unavailable", locale: "en", message: "voice session unavailable", leaked: secret,
			cause:  fmt.Errorf("provider body %s: %w", secret, voicesession.ErrSessionUnavailable),
			status: http.StatusServiceUnavailable, code: errs.SessionUnavailable, logged: false,
		},
		{
			name: "session unavailable indonesian", locale: "id", message: "sesi suara tidak tersedia", leaked: secret,
			cause:  fmt.Errorf("provider body %s: %w", secret, voicesession.ErrSessionUnavailable),
			status: http.StatusServiceUnavailable, code: errs.SessionUnavailable, logged: false,
		},
		{
			name: "technical failure", locale: "en", message: "internal error", leaked: "sql diagnostic",
			cause:  errs.Technical{Cause: errors.New("sql diagnostic")},
			status: http.StatusInternalServerError, code: errs.InternalError, logged: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			operation := mocks.NewNegotiateOperation(t)
			operation.On("Execute", mock.Anything, "offer-sdp").Return("", tt.cause)
			logged := false
			var loggedErr error
			handler := voicesessionhandler.Handler{
				Negotiate: operation,
				LogError: func(err error) {
					logged = true
					loggedErr = err
				},
			}
			recorder := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodPost, "/voice-sessions", strings.NewReader(`{"sdp":"offer-sdp"}`))
			request.Header.Set("Accept-Language", tt.locale)
			handler.ServeHTTP(recorder, request)
			requireJSONStatus(t, recorder, tt.status)
			requireErrorEnvelope(t, recorder.Body.Bytes(), "error", tt.code, tt.message, tt.leaked)
			require.Equal(t, tt.logged, logged)
			if tt.logged {
				require.ErrorIs(t, loggedErr, tt.cause)
			}
		})
	}
}
