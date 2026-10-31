package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/SALT-Indonesia/salt-pkg/logmanager"
	"github.com/rfanazhari/amer-prototype/infrastructure/runtime"
)

func main() {
	logs := logmanager.NewApplication(logmanager.WithAppName("amer-prototype"), logmanager.WithService("amer-prototype"), logmanager.WithSkipHeaders(), logmanager.WithMaskingConfig([]logmanager.MaskingConfig{
		{FieldPattern: "authorization", Type: logmanager.FullMask}, {FieldPattern: "cookie", Type: logmanager.FullMask}, {FieldPattern: "password", Type: logmanager.FullMask}, {FieldPattern: "token", Type: logmanager.FullMask}, {FieldPattern: "email", Type: logmanager.FullMask}, {FieldPattern: "api_key", Type: logmanager.FullMask},
	}))
	address := os.Getenv("HTTP_ADDRESS")
	if address == "" {
		address = ":8080"
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	shutdownStarted := make(chan time.Time, 1)
	finished := make(chan struct{})
	go runtime.WatchShutdown(ctx, shutdownStarted, finished, func() {
		// Logging is best effort here; blocked logging must not extend the deadline.
		go func() {
			txn := logs.Start("", "shutdown deadline exceeded; cleanup incomplete", logmanager.TxnTypeOther)
			txn.NoticeError(context.DeadlineExceeded)
			txn.End()
		}()
		os.Exit(1)
	})
	if err := runtime.Run(ctx, runtime.Config{DatabaseURL: os.Getenv("DATABASE_URL"), Address: address, ShutdownStarted: shutdownStarted}, logs); err != nil {
		txn := logs.Start("", "service failure", logmanager.TxnTypeOther)
		txn.NoticeError(err)
		txn.End()
		os.Exit(1)
	}
	close(finished)
}
