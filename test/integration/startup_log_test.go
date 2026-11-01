package integration_test

import (
	"context"
	"net"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/SALT-Indonesia/salt-pkg/logmanager"
	"github.com/rfanazhari/amer-prototype/infrastructure/database/schema"
	appruntime "github.com/rfanazhari/amer-prototype/infrastructure/runtime"
	"github.com/stretchr/testify/require"
)

// startupLogCase specifies logs for successful startup or a listener failure after database setup.
type startupLogCase struct {
	name       string
	failListen bool
	wantErr    bool
	messages   []string
}

// TestStartupLogsReportReadinessWithoutSecrets exercises lifecycle logging through real composition.
func TestStartupLogsReportReadinessWithoutSecrets(t *testing.T) {
	store(t)
	tests := []startupLogCase{
		{name: "ready", messages: []string{"service starting", "database connected and schema compatible", "administrative bootstrap complete", "http server ready"}},
		{name: "listener failure", failListen: true, wantErr: true, messages: []string{"service starting", "database connected and schema compatible", "service startup failed"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logs := logmanager.NewTestableApplication(logmanager.WithAppName("startup-test"), logmanager.WithService("amer-prototype"))
			ready := make(chan net.Addr, 1)
			cfg := lifecycleConfig(databaseURL(t, "startup-logs"), "127.0.0.1:0", ready)
			cfg.AccessBootstrapAdminPassword = "startup-password-must-stay-private"
			if tt.failListen {
				listener, err := net.Listen("tcp", "127.0.0.1:0")
				require.NoError(t, err)
				defer func() { require.NoError(t, listener.Close()) }()
				cfg.Address = listener.Addr().String()
			}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			result := make(chan error, 1)
			go func() { result <- appruntime.Run(ctx, cfg, logs.Application) }()
			var address net.Addr
			if !tt.wantErr {
				select {
				case address = <-ready:
					cancel()
				case err := <-result:
					t.Fatalf("startup failed: %v", err)
				case <-time.After(10 * time.Second):
					t.Fatal("startup timed out")
				}
			}
			select {
			case err := <-result:
				if tt.wantErr {
					require.Error(t, err)
				} else {
					require.NoError(t, err)
				}
			case <-time.After(6 * time.Second):
				t.Fatal("runtime did not terminate")
			}
			messages := []string{}
			for _, entry := range logs.GetLoggedEntries() {
				if entry.Message != "" {
					messages = append(messages, entry.Message)
				}
				line, err := entry.String()
				require.NoError(t, err)
				for _, secret := range []string{cfg.DatabaseURL, cfg.AccessBootstrapAdminPassword, cfg.AccessCSRFSecret, cfg.AccessBootstrapAdminEmail} {
					require.False(t, strings.Contains(line, secret), "startup log leaked protected configuration")
				}
				if entry.Message == "http server ready" {
					require.Equal(t, address.String(), entry.Data["address"])
					require.Equal(t, "in_process", entry.Data["event_queue"])
					require.Equal(t, strconv.Itoa(schema.ReleaseVersion), entry.Data["schema_version"])
				}
			}
			require.Equal(t, tt.messages, messages)
		})
	}
}
