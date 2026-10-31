package runtime_test

import (
	"context"
	"os"
	"os/exec"
	"testing"
	"time"

	appruntime "github.com/rfanazhari/amer-prototype/infrastructure/runtime"
	"github.com/stretchr/testify/require"
)

// TestShutdownWatchdogReportsExpiredDeadline checks that an inherited expired deadline is not reset.
func TestShutdownWatchdogReportsExpiredDeadline(t *testing.T) {
	started := make(chan time.Time, 1)
	started <- time.Now().Add(-appruntime.ShutdownBudget)
	expired := make(chan struct{})
	done := make(chan struct{})
	go func() {
		defer close(done)
		appruntime.WatchShutdown(context.Background(), started, make(chan struct{}), func() { close(expired) })
	}()
	select {
	case <-expired:
	case <-time.After(time.Second):
		t.Fatal("expired cleanup was not reported")
	}
	<-done
}

// TestShutdownWatchdogTerminatesBlockedProcess checks the configured process deadline against a subprocess whose work never completes.
func TestShutdownWatchdogTerminatesBlockedProcess(t *testing.T) {
	if os.Getenv("AMER_SHUTDOWN_HELPER") == "1" {
		started := make(chan time.Time, 1)
		started <- time.Now()
		go appruntime.WatchShutdown(context.Background(), started, make(chan struct{}), func() { os.Exit(17) })
		select {}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	command := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestShutdownWatchdogTerminatesBlockedProcess$")
	command.Env = append(os.Environ(), "AMER_SHUTDOWN_HELPER=1")
	beginning := time.Now()
	err := command.Run()
	var exitErr *exec.ExitError
	require.ErrorAs(t, err, &exitErr)
	require.NoError(t, ctx.Err())
	require.Equal(t, 17, exitErr.ExitCode())
	require.GreaterOrEqual(t, time.Since(beginning), appruntime.ShutdownBudget)
}

// TestShutdownWatchdogStopsAfterCompletedCleanup checks that completed cleanup disarms forced process termination.
func TestShutdownWatchdogStopsAfterCompletedCleanup(t *testing.T) {
	finished := make(chan struct{})
	close(finished)
	called := false
	appruntime.WatchShutdown(context.Background(), make(chan time.Time), finished, func() { called = true })
	require.False(t, called)
}
