package realtimecall_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/SALT-Indonesia/salt-pkg/logmanager"
	"github.com/rfanazhari/amer-prototype/application/usecase/voicesession"
	"github.com/rfanazhari/amer-prototype/infrastructure/service/realtimecall"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testKey          = "server-key"
	testModel        = "gpt-realtime-2.1"
	testVoice        = "marin"
	testOffer        = "v=0 offer"
	testAnswer       = "v=0 answer"
	testInstructions = "interview in Indonesian"
	providerSecret   = "secret provider dump"
)

// capturedCall records one inbound Live session request observed by the local server.
type capturedCall struct {
	method        string
	authorization string
	contentType   string
	sdp           string
	transportType string
	session       map[string]any
}

// adapterCase defines Exchange inputs and the expected local-provider interaction.
type adapterCase struct {
	name           string
	key            string
	status         int
	responseBody   string
	want           string
	wantErr        error
	wantCalls      int32
	hideBody       bool
	closedServer   bool
	inspectRequest bool
}

// tracedContext returns a context that carries the logmanager transaction clientmanager requires.
func tracedContext(t *testing.T) context.Context {
	t.Helper()
	logs := logmanager.NewApplication(
		logmanager.WithAppName("realtimecall-test"),
		logmanager.WithSkipHeaders(),
		logmanager.WithMaskingConfig([]logmanager.MaskingConfig{{
			FieldPattern: "authorization",
			Type:         logmanager.FullMask,
		}}),
	)
	txn := logs.Start("", "realtime setup", logmanager.TxnTypeOther)
	t.Cleanup(txn.End)
	return txn.ToContext(context.Background())
}

// TestAdapterExchange checks one authorized Live session call, answer passthrough, and safe failures.
func TestAdapterExchange(t *testing.T) {
	tests := []adapterCase{
		{
			name:           "posts one authorized live session and returns the answer",
			key:            testKey,
			status:         http.StatusCreated,
			responseBody:   `{"transport":{"sdp":"` + testAnswer + `","type":"webrtc"}}`,
			want:           testAnswer,
			wantCalls:      1,
			inspectRequest: true,
		},
		{
			name:         "maps a provider error without exposing the body",
			key:          testKey,
			status:       http.StatusInternalServerError,
			responseBody: providerSecret,
			wantErr:      voicesession.ErrSessionUnavailable,
			wantCalls:    1,
			hideBody:     true,
		},
		{
			name:      "does not dial when the key is empty",
			key:       "",
			wantErr:   voicesession.ErrSessionUnavailable,
			wantCalls: 0,
		},
		{
			name:         "maps a transport failure",
			key:          testKey,
			wantErr:      voicesession.ErrSessionUnavailable,
			closedServer: true,
			hideBody:     true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var calls atomic.Int32
			var got capturedCall
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				got.method = r.Method
				got.authorization = r.Header.Get("Authorization")
				got.contentType = r.Header.Get("Content-Type")
				var payload map[string]any
				require.NoError(t, json.NewDecoder(r.Body).Decode(&payload))
				transport, _ := payload["transport"].(map[string]any)
				got.sdp, _ = transport["sdp"].(string)
				got.transportType, _ = transport["type"].(string)
				got.session, _ = payload["session"].(map[string]any)
				w.WriteHeader(tt.status)
				_, err := io.WriteString(w, tt.responseBody)
				require.NoError(t, err)
			}))
			url := server.URL
			if tt.closedServer {
				server.Close()
			} else {
				t.Cleanup(server.Close)
			}
			adapter := realtimecall.Adapter{
				APIKey: tt.key,
				Model:  testModel,
				Voice:  testVoice,
				URL:    url,
			}
			answer, err := adapter.Exchange(tracedContext(t), testOffer, testInstructions)
			assert.Equal(t, tt.wantCalls, calls.Load())
			if tt.wantErr != nil {
				require.Error(t, err)
				assert.ErrorIs(t, err, tt.wantErr)
				assert.Empty(t, answer)
				assert.NotContains(t, err.Error(), testKey)
				if tt.hideBody {
					assert.NotContains(t, err.Error(), providerSecret)
				}
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, answer)
			if !tt.inspectRequest {
				return
			}
			assert.Equal(t, http.MethodPost, got.method)
			assert.Contains(t, got.contentType, "application/json")
			assert.Equal(t, "Bearer "+testKey, got.authorization)
			assert.Equal(t, testOffer+"\r\n", got.sdp)
			assert.Equal(t, "webrtc", got.transportType)
			assert.Equal(t, "gpt-live-1", got.session["model"])
			assert.Equal(t, testInstructions, got.session["instructions"])
			audio, ok := got.session["audio"].(map[string]any)
			require.True(t, ok)
			output, ok := audio["output"].(map[string]any)
			require.True(t, ok)
			assert.Equal(t, testVoice, output["voice"])
		})
	}
}
