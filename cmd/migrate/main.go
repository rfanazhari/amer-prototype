// Command migrate prepares a database explicitly; it never starts the application.
package main

import (
	"context"
	"database/sql"
	"fmt"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/rfanazhari/amer-prototype/infrastructure/database/schema"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"
)

// run reads credentials only from the environment and emits safe operation outcomes.
func run() int {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "usage: migrate status|up|baseline|down")
		return 1
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()
	url := os.Getenv("MIGRATION_DATABASE_URL")
	if url == "" {
		fmt.Fprintln(os.Stderr, "MIGRATION_DATABASE_URL is required")
		return 1
	}
	db, err := sql.Open("pgx", url)
	if err != nil {
		fmt.Fprintln(os.Stderr, "migration connection configuration failed")
		return 1
	}
	defer func() { _ = db.Close() }()
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(0)
	version, _ := strconv.ParseUint(os.Getenv("MIGRATION_VERSION"), 10, 32)
	steps, _ := strconv.Atoi(os.Getenv("MIGRATION_STEPS"))
	directory, err := os.Getwd()
	if err != nil {
		return 1
	}
	v, dirty, err := (schema.Runner{DB: db, Directory: directory + "/migrations/pgsql"}).Execute(ctx, schema.Options{Action: os.Args[1], Version: uint(version), Steps: steps, Disposable: os.Getenv("MIGRATION_DISPOSABLE") == "yes"})
	if err != nil {
		fmt.Fprintln(os.Stderr, "migration failed; inspect schema and dirty state before reviewed recovery")
		return 1
	}
	fmt.Printf("version=%d dirty=%t\n", v, dirty)
	return 0
}

// main sets the command exit status after deferred resources have closed.
func main() { os.Exit(run()) }
