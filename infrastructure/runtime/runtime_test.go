package runtime_test

import (
	"context"
	"strings"
	"testing"

	"github.com/SALT-Indonesia/salt-pkg/logmanager"
	appruntime "github.com/rfanazhari/amer-prototype/infrastructure/runtime"
	"github.com/stretchr/testify/require"
)

// startupCase defines invalid startup configuration or cancellation before acquisition.
type startupCase struct {
	name           string
	cfg            appruntime.Config
	cancel         bool
	wantErr        bool
	requiredConfig bool
	forbidKeyError bool
}

// TestStartupValidationAndCancellation checks required configuration and cancellation before resource acquisition.
func TestStartupValidationAndCancellation(t *testing.T) {
	logs := logmanager.NewApplication(logmanager.WithAppName("runtime-test"))
	tests := []startupCase{
		{name: "missing database", cfg: appruntime.Config{Address: "127.0.0.1:0"}, wantErr: true, requiredConfig: true},
		{name: "missing address", cfg: appruntime.Config{DatabaseURL: "unused"}, wantErr: true, requiredConfig: true},
		{name: "empty provider key is not required", cfg: appruntime.Config{Address: "127.0.0.1:0", OpenAIAPIKey: ""}, wantErr: true, requiredConfig: true, forbidKeyError: true},
		{name: "cancelled before acquisition", cfg: appruntime.Config{DatabaseURL: "unused", Address: "127.0.0.1:0"}, cancel: true, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			if tt.cancel {
				cancel()
			}
			err := appruntime.Run(ctx, tt.cfg, logs)
			if tt.wantErr {
				require.Error(t, err)
			}
			if tt.requiredConfig {
				require.ErrorContains(t, err, "database URL, address and logging are required")
			}
			if tt.forbidKeyError {
				lower := strings.ToLower(err.Error())
				require.NotContains(t, lower, "openai")
				require.NotContains(t, lower, "api key")
				require.NotContains(t, lower, "provider")
			}
			if tt.cancel {
				require.ErrorIs(t, err, context.Canceled)
			}
		})
	}
}
