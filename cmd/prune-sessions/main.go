// Command prune-sessions removes administrative sessions long past expiry or revocation; an external scheduler runs it.
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/rfanazhari/amer-prototype/infrastructure/runtime"
)

// run reads the runtime database URL from the environment and reports only the removed row count.
func run() int {
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		fmt.Fprintln(os.Stderr, "DATABASE_URL is required")
		return 1
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()
	removed, err := runtime.PruneAdministrativeSessions(ctx, url)
	fmt.Printf("removed=%d\n", removed)
	if err != nil {
		fmt.Fprintln(os.Stderr, "session cleanup stopped early; rerun to continue")
		return 1
	}
	return 0
}

// main sets the command exit status after deferred resources have closed.
func main() { os.Exit(run()) }
