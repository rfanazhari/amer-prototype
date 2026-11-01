// Package schema owns explicit versioned database preparation, independently of application startup.
package schema

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/golang-migrate/migrate/v4"
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
func (r Runner) Execute(ctx context.Context, o Options) (version int, dirty bool, resultErr error) {
	ctx, cancel := context.WithTimeout(ctx, 59*time.Second)
	defer cancel()
	switch o.Action {
	case "status":
		if r.DB == nil {
			return 0, false, errors.New("database required")
		}
		var v int
		var dirty bool
		err := r.DB.QueryRowContext(ctx, "SELECT version, dirty FROM public.schema_migrations").Scan(&v, &dirty)
		return v, dirty, err
	case "up":
		if o.Version < 1 || o.Version > ReleaseVersion {
			return 0, false, fmt.Errorf("explicit supported target 1 through %d required", ReleaseVersion)
		}
	case "down":
		if o.Steps < 1 || o.Steps > ReleaseVersion || !o.Disposable {
			return 0, false, errors.New("down requires steps and disposable confirmation")
		}
	case "baseline":
	default:
		return 0, false, errors.New("unsupported migration operation")
	}
	if r.DB == nil {
		return 0, false, errors.New("database required")
	}
	conn, err := r.DB.Conn(ctx)
	if err != nil {
		return 0, false, err
	}
	defer func() { resultErr = errors.Join(resultErr, conn.Close()) }()
	d := &driver{conn: conn, ctx: ctx}
	if err = d.Lock(); err != nil {
		return 0, false, err
	}
	defer func() { resultErr = errors.Join(resultErr, d.cleanup()) }()
	if err = d.preflight(o.Action == "baseline"); err != nil {
		return 0, false, err
	}
	if o.Action == "baseline" {
		return d.baseline()
	}
	if _, err = conn.ExecContext(ctx, metadataDDL); err != nil {
		return 0, false, err
	}
	src, err := artifacts(r.Directory)
	if err != nil {
		return 0, false, err
	}
	defer func() { resultErr = errors.Join(resultErr, src.Close()) }()
	m, err := migrate.NewWithInstance("file", src, "postgres", d)
	if err != nil {
		return 0, false, err
	}
	m.LockTimeout = 5 * time.Second
	v, dirty, err := d.Version()
	if err != nil {
		return v, dirty, err
	}
	if dirty || v > ReleaseVersion || v == 0 || v < -1 {
		return v, dirty, errors.New("dirty or unsupported version")
	}
	if o.Action == "up" {
		if v > int(o.Version) {
			return v, dirty, errors.New("up cannot downgrade")
		}
		err = m.Migrate(o.Version)
	} else {
		err = m.Steps(-o.Steps)
	}
	if err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return v, dirty, err
	}
	return d.Version()
}
