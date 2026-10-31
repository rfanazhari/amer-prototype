package runtime_test

import (
	"context"
	"testing"

	"github.com/SALT-Indonesia/salt-pkg/logmanager"
	appruntime "github.com/rfanazhari/amer-prototype/infrastructure/runtime"
	"github.com/stretchr/testify/require"
)

// startupCase defines invalid startup configuration or cancellation before acquisition.
type startupCase struct {
	name    string
	cfg     appruntime.Config
	cancel  bool
	wantErr bool
}

// TestStartupValidationAndCancellation checks required configuration and cancellation before resource acquisition.
func TestStartupValidationAndCancellation(t *testing.T) {
	logs := logmanager.NewApplication(logmanager.WithAppName("runtime-test"))
	tests := []startupCase{
		{name: "missing database", cfg: appruntime.Config{Address: "127.0.0.1:0"}, wantErr: true},
		{name: "missing address", cfg: appruntime.Config{DatabaseURL: "unused"}, wantErr: true},
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
			if tt.cancel {
				require.ErrorIs(t, err, context.Canceled)
			}
		})
	}
}
