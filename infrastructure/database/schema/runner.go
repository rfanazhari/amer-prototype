// Package schema owns explicit versioned database preparation, independently of application startup.
package schema

import (
	"context"
	"database/sql"
	"errors"
)

// Options specifies an explicit migration operation without database credentials.
type Options struct {
	Action     string
	Version    uint
	Steps      int
	Disposable bool
}

// Runner operates on a caller-owned migration database.
type Runner struct {
	DB        *sql.DB
	Directory string
}

// Execute validates and performs a bounded explicit schema operation.
func (r Runner) Execute(ctx context.Context, o Options) (int, bool, error) {
	return 0, false, errors.New("not implemented")
}
