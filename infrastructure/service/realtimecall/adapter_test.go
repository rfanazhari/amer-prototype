package realtimecall_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

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

// capturedCall records one inbound Realtime calls request observed by the local server.
type capturedCall struct {
	method        string
	authorization string
	sdp           string
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

// TestAdapterExchange checks one authorized multipart call, answer passthrough, and safe failures.
func TestAdapterExchange(t *testing.T) {
	tests := []adapterCase{
		{
			name:           "posts one authorized multipart call and returns the answer",
			key:            testKey,
			status:         http.StatusCreated,
			responseBody:   testAnswer,
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
				require.NoError(t, r.ParseMultipartForm(1<<20))
				got.sdp = r.FormValue("sdp")
				require.NoError(t, json.Unmarshal([]byte(r.FormValue("session")), &got.session))
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
			answer, err := adapter.Exchange(context.Background(), testOffer, testInstructions)
			assert.Equal(t, tt.wantCalls, calls.Load())
			if tt.wantErr != nil {
				require.Error(t, err)
				assert.ErrorIs(t, err, tt.wantErr)
				assert.Empty(t, answer)
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
			assert.Equal(t, "Bearer "+testKey, got.authorization)
			assert.Equal(t, testOffer, got.sdp)
			assert.Equal(t, "realtime", got.session["type"])
			assert.Equal(t, testModel, got.session["model"])
			assert.Equal(t, testVoice, got.session["voice"])
			assert.Equal(t, testInstructions, got.session["instructions"])
			audio, ok := got.session["audio"].(map[string]any)
			require.True(t, ok)
			input, ok := audio["input"].(map[string]any)
			require.True(t, ok)
			turnDetection, ok := input["turn_detection"].(map[string]any)
			require.True(t, ok)
			assert.Equal(t, "server_vad", turnDetection["type"])
			assert.Equal(t, true, turnDetection["create_response"])
			assert.Equal(t, true, turnDetection["interrupt_response"])
		})
	}
}
