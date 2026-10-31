package voicepage_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/rfanazhari/amer-prototype/delivery/http/voicepage"
	"github.com/rfanazhari/amer-prototype/pkg/errs"
	"github.com/stretchr/testify/require"
)

// documentCase is one GET /voice-session assertion against the served HTML asset.
type documentCase struct {
	name     string
	contains []string
	absent   []string
}

// methodCase is one unsupported method on the page path.
type methodCase struct {
	name, method, code, message string
	status                      int
}

// scriptCase is one source-level lifecycle rule in the served page script.
type scriptCase struct {
	name string
}

// envelopeView is the decoded public JSON envelope asserted by method-error tests.
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

// getPage issues GET /voice-session against the page handler.
func getPage(t *testing.T) *httptest.ResponseRecorder {
	t.Helper()
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/voice-session", nil)
	voicepage.Handler{}.ServeHTTP(recorder, request)
	return recorder
}

// pageBody returns the HTML document served for GET /voice-session.
func pageBody(t *testing.T) string {
	t.Helper()
	recorder := getPage(t)
	require.Equal(t, http.StatusOK, recorder.Code)
	require.Contains(t, recorder.Header().Get("Content-Type"), "text/html")
	return recorder.Body.String()
}

// scriptFrom extracts the first inline script from an HTML document.
func scriptFrom(t *testing.T, html string) string {
	t.Helper()
	start := strings.Index(html, "<script>")
	end := strings.Index(html, "</script>")
	require.GreaterOrEqual(t, start, 0)
	require.Greater(t, end, start)
	return html[start+len("<script>") : end]
}

// functionBody returns the source of a named function, including nested braces.
func functionBody(t *testing.T, script, name string) string {
	t.Helper()
	marker := "function " + name + "("
	start := strings.Index(script, marker)
	require.GreaterOrEqual(t, start, 0, "missing function %s", name)
	open := strings.Index(script[start:], "{")
	require.GreaterOrEqual(t, open, 0)
	open += start
	depth := 0
	for i := open; i < len(script); i++ {
		switch script[i] {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return script[open : i+1]
			}
		}
	}
	require.Fail(t, "unclosed function "+name)
	return ""
}

// indexOf requires needle to appear in haystack and returns its offset.
func indexOf(t *testing.T, haystack, needle string) int {
	t.Helper()
	idx := strings.Index(haystack, needle)
	require.GreaterOrEqual(t, idx, 0, "missing %q", needle)
	return idx
}

// TestVoicePageServesDocument checks the FSD controls, statuses, synthetic notice, and key absence.
func TestVoicePageServesDocument(t *testing.T) {
	tests := []documentCase{
		{
			name: "controls statuses and synthetic notice",
			contains: []string{
				"Start Conversation",
				"Stop Conversation",
				"Idle",
				"Connecting",
				"Connected",
				"Disconnected",
				"Error",
				"synthetic",
			},
			absent: []string{
				"OPENAI_API_KEY",
				"Authorization",
				"Bearer ",
				"sk-",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			html := pageBody(t)
			lower := strings.ToLower(html)
			for _, needle := range tt.contains {
				require.Contains(t, html, needle)
			}
			for _, needle := range tt.absent {
				require.NotContains(t, lower, strings.ToLower(needle))
			}
		})
	}
}

// TestVoicePageScriptLifecycle checks permission-before-setup, one peer connection, opening event, and release.
func TestVoicePageScriptLifecycle(t *testing.T) {
	tests := []scriptCase{
		{name: "microphone then setup on one peer connection"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			html := pageBody(t)
			script := scriptFrom(t, html)
			start := functionBody(t, script, "startConversation")
			stop := functionBody(t, script, "stopConversation")
			release := functionBody(t, script, "releaseSession")
			permission := functionBody(t, script, "handlePermissionDenied")
			setupFail := functionBody(t, script, "handleSetupFailure")
			drop := functionBody(t, script, "handleDrop")
			attachPlayback := functionBody(t, script, "attachPlayback")

			require.Less(t, indexOf(t, start, "getUserMedia"), indexOf(t, start, "/voice-sessions"))
			require.Equal(t, 1, strings.Count(script, "new RTCPeerConnection"))
			require.Equal(t, 1, strings.Count(script, `"type":"response.create"`))
			require.Contains(t, functionBody(t, script, "sendOpeningResponse"), `"type":"response.create"`)
			require.Contains(t, start, "sendOpeningResponse")
			require.NotContains(t, attachPlayback, "response.create")
			require.NotContains(t, attachPlayback, "/voice-sessions")
			require.Contains(t, start, "createDataChannel")
			require.Contains(t, start, "oai-events")

			require.Contains(t, release, ".stop(")
			require.Contains(t, release, ".close(")
			require.Contains(t, stop, "releaseSession")
			require.Contains(t, permission, "releaseSession")
			require.Contains(t, setupFail, "releaseSession")
			require.Contains(t, drop, "releaseSession")
			require.NotContains(t, permission, "startConversation(")
			require.NotContains(t, setupFail, "startConversation(")
			require.NotContains(t, drop, "startConversation(")
			require.NotContains(t, drop, "new RTCPeerConnection")
			require.NotContains(t, script, "setInterval")
		})
	}
}

// terminalCase is one source rule that keeps a finished attempt on its designed status.
type terminalCase struct {
	name string
}

// TestVoicePageTerminalStates checks setup, stop, gathering failure, and a recoverable ICE state.
func TestVoicePageTerminalStates(t *testing.T) {
	tests := []terminalCase{
		{name: "local release does not replace the designed status"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			script := scriptFrom(t, pageBody(t))
			start := functionBody(t, script, "startConversation")
			stop := functionBody(t, script, "stopConversation")
			release := functionBody(t, script, "releaseSession")
			permission := functionBody(t, script, "handlePermissionDenied")
			setupFail := functionBody(t, script, "handleSetupFailure")
			drop := functionBody(t, script, "handleDrop")
			gather := functionBody(t, script, "waitForIce")

			require.Less(t, indexOf(t, setupFail, "stopping = true"), indexOf(t, setupFail, "releaseSession"))
			require.Less(t, indexOf(t, permission, "stopping = true"), indexOf(t, permission, "releaseSession"))
			require.Less(t, indexOf(t, drop, "stopping = true"), indexOf(t, drop, "releaseSession"))
			require.Contains(t, stop, "stopping = true")
			require.Contains(t, stop, `setStatus("Idle")`)
			require.NotContains(t, stop, "stopping = false")
			require.Less(t, indexOf(t, release, "dataChannel = null"), indexOf(t, release, ".close("))
			require.Less(t, indexOf(t, release, "peerConnection = null"), indexOf(t, release, "connection.close"))
			require.Contains(t, start, "event.target !== dataChannel")
			require.Contains(t, start, `state === "failed"`)
			require.NotContains(t, start, `"disconnected"`)
			require.Contains(t, start, `err.name === "IceGatheringError"`)
			require.Contains(t, gather, "IceGatheringError")
			require.Contains(t, gather, "iceGatheringLimitMs")
			require.Contains(t, gather, "reject(")
			require.NotContains(t, script, "setInterval")
		})
	}
}

// TestVoicePageRejectsPost checks that POST /voice-session is METHOD_NOT_ALLOWED.
func TestVoicePageRejectsPost(t *testing.T) {
	tests := []methodCase{
		{name: "post", method: http.MethodPost, status: http.StatusMethodNotAllowed, code: errs.MethodNotAllowed, message: "method not allowed"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			request := httptest.NewRequest(tt.method, "/voice-session", nil)
			voicepage.Handler{}.ServeHTTP(recorder, request)
			require.Equal(t, tt.status, recorder.Code)
			require.Equal(t, "application/json", recorder.Header().Get("Content-Type"))
			var keys map[string]json.RawMessage
			require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &keys))
			var view envelopeView
			require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &view))
			require.ElementsMatch(t, []string{"status", "data", "error"}, objectKeys(t, recorder.Body.Bytes()))
			require.Equal(t, "error", view.Status)
			require.JSONEq(t, `{}`, string(keys["data"]))
			require.ElementsMatch(t, []string{"code", "message"}, objectKeys(t, keys["error"]))
			require.Equal(t, tt.code, view.Error.Code)
			require.Equal(t, tt.message, view.Error.Message)
		})
	}
}
