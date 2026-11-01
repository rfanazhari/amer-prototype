package schema

import (
	"context"
	"database/sql"
	sqldriver "database/sql/driver"
	"errors"
	"io"
	"time"

	"github.com/golang-migrate/migrate/v4/database"
)

const metadataDDL = `CREATE TABLE IF NOT EXISTS public.schema_migrations (version bigint NOT NULL PRIMARY KEY, dirty boolean NOT NULL)`

// driver binds golang-migrate's context-free API to one bounded command context and connection.
type driver struct {
	conn   *sql.Conn
	ctx    context.Context
	locked bool
	lockID string
}

// Open rejects URL-based construction; credentials are owned by the command.
func (d *driver) Open(string) (database.Driver, error) {
	return nil, errors.New("use explicit connection")
}

// Close leaves the caller-owned connection to Execute.
func (d *driver) Close() error { return nil }

// Lock acquires the tool-compatible advisory lock with a five-second server wait.
func (d *driver) Lock() error {
	if d.locked {
		return nil
	}
	var name string
	if err := d.conn.QueryRowContext(d.ctx, "SELECT current_database()").Scan(&name); err != nil {
		return err
	}
	id, err := database.GenerateAdvisoryLockId(name, "public", "schema_migrations")
	if err != nil {
		return err
	}
	d.lockID = id
	if _, err = d.conn.ExecContext(d.ctx, "SET lock_timeout = '5s'"); err != nil {
		return err
	}
	if _, err = d.conn.ExecContext(d.ctx, "SELECT pg_advisory_lock($1)", id); err != nil {
		return err
	}
	d.locked = true
	return nil
}

// Unlock releases coordination using the active command context.
func (d *driver) Unlock() error {
	if !d.locked {
		return nil
	}
	_, err := d.conn.ExecContext(d.ctx, "SELECT pg_advisory_unlock($1)", d.lockID)
	if err == nil {
		d.locked = false
	}
	return err
}

// cleanup rolls back failed explicit SQL transactions and releases locks with a bounded independent context.
func (d *driver) cleanup() error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	_, rollbackErr := d.conn.ExecContext(ctx, "ROLLBACK")
	var unlockErr error
	if d.locked {
		_, unlockErr = d.conn.ExecContext(ctx, "SELECT pg_advisory_unlock($1)", d.lockID)
	}
	err := errors.Join(rollbackErr, unlockErr)
	if err != nil {
		_ = d.conn.Raw(func(any) error { return sqldriver.ErrBadConn })
	}
	return err
}

// Run executes an immutable SQL artifact, including its explicit transaction, under the command deadline.
func (d *driver) Run(r io.Reader) error {
	b, err := io.ReadAll(r)
	if err != nil {
		return err
	}
	_, err = d.conn.ExecContext(d.ctx, string(b))
	return err
}

// SetVersion atomically replaces tool metadata, leaving dirty state on SQL failure.
func (d *driver) SetVersion(v int, dirty bool) error {
	tx, err := d.conn.BeginTx(d.ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	if _, err = tx.ExecContext(d.ctx, "DELETE FROM public.schema_migrations"); err != nil {
		return err
	}
	if v >= 0 || dirty {
		if _, err = tx.ExecContext(d.ctx, "INSERT INTO public.schema_migrations (version, dirty) VALUES ($1, $2)", v, dirty); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// Version refuses ambiguous metadata instead of selecting an arbitrary row.
func (d *driver) Version() (int, bool, error) {
	rows, err := d.conn.QueryContext(d.ctx, "SELECT version, dirty FROM public.schema_migrations")
	if err != nil {
		return 0, false, err
	}
	defer func() { _ = rows.Close() }()
	v := -1
	dirty := false
	count := 0
	for rows.Next() {
		count++
		if err = rows.Scan(&v, &dirty); err != nil {
			return 0, false, err
		}
	}
	if err = rows.Err(); err != nil {
		return 0, false, err
	}
	if count > 1 {
		return 0, false, errors.New("foreign migration metadata")
	}
	return v, dirty, nil
}

// Drop is intentionally unavailable; explicit guarded down is the only destructive operation.
func (d *driver) Drop() error { return errors.New("drop is disabled") }
