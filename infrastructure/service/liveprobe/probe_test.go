package liveprobe_test

import (
	"testing"

	"github.com/rfanazhari/amer-prototype/infrastructure/service/liveprobe"
	"github.com/stretchr/testify/require"
)

// setupCase describes provider responses and the identity contract required for attachment.
type setupCase struct {
	name    string
	body    string
	wantErr bool
}

// TestDecodeSetup requires both session identity and an SDP answer, without accepting partial setup.
func TestDecodeSetup(t *testing.T) {
	for _, tt := range []setupCase{
		{name: "complete", body: `{"session":{"id":"live_test"},"transport":{"sdp":"v=0\r\n","type":"webrtc"}}`},
		{name: "missing identity", body: `{"transport":{"sdp":"v=0"}}`, wantErr: true},
		{name: "missing answer", body: `{"session":{"id":"live_test"}}`, wantErr: true},
		{name: "unsafe identity", body: `{"session":{"id":"../secret"},"transport":{"sdp":"v=0","type":"webrtc"}}`, wantErr: true},
		{name: "malformed", body: `{`, wantErr: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			got, err := liveprobe.DecodeSetup([]byte(tt.body))
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, "live_test", got.SessionID)
			require.Equal(t, "v=0\r\n", got.SDP)
		})
	}
}

// TestShapeDoesNotExposeContent checks diagnostics omit arbitrary transcript and credential values.
func TestShapeDoesNotExposeContent(t *testing.T) {
	got, err := liveprobe.Shape([]byte(`{"type":"session.output_transcript.delta","delta":"private answer","audio":"secret audio"}`))
	require.NoError(t, err)
	require.Contains(t, got, "session.output_transcript.delta")
	require.NotContains(t, got, "private answer")
	require.NotContains(t, got, "secret audio")
}
