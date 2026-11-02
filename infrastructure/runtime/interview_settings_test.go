package runtime_test

import (
	"testing"

	appruntime "github.com/rfanazhari/amer-prototype/infrastructure/runtime"
	"github.com/stretchr/testify/require"
)

// TestInterviewSettings validates optional Live defaults and refuses malformed endpoint overrides.
func TestInterviewSettings(t *testing.T) {
	// testCase names one startup configuration boundary.
	type testCase struct {
		name    string
		cfg     appruntime.Config
		wantErr bool
	}
	tests := []testCase{
		{name: "empty settings select defaults"},
		{name: "fake provider endpoints", cfg: appruntime.Config{OpenAILiveURL: "http://127.0.0.1:9000/live/sessions", OpenAILiveAttachURL: "ws://127.0.0.1:9000/live/sessions"}},
		{name: "missing host", cfg: appruntime.Config{OpenAILiveURL: "/live/sessions"}, wantErr: true},
		{name: "wrong create scheme", cfg: appruntime.Config{OpenAILiveURL: "file://localhost/key"}, wantErr: true},
		{name: "wrong attach scheme", cfg: appruntime.Config{OpenAILiveAttachURL: "http://localhost"}, wantErr: true},
		{name: "credentials in URL", cfg: appruntime.Config{OpenAILiveURL: "https://user:pass@example.test"}, wantErr: true},
		{name: "blank model", cfg: appruntime.Config{OpenAILiveModel: " "}, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := appruntime.ValidateInterviewSettings(tt.cfg)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
