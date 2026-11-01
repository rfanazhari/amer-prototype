package schema

import (
	"embed"
	"encoding/json"
	"errors"
	"reflect"
)

// catalogAssets freezes independently observed legacy catalog facts.
//
//go:embed catalog.sql tasks.json administrative.json metadata.json checksums.json
var catalogAssets embed.FS

// preflight rejects foreign schemas/metadata before any tool-owned DDL.
func (d *driver) preflight(adopting bool) error {
	var foreign int
	err := d.conn.QueryRowContext(d.ctx, `SELECT count(*) FROM information_schema.tables WHERE table_schema='public' AND table_name NOT IN ('tasks','administrative_accounts','administrative_sessions','schema_migrations','agent_instances','agent_drafts','agent_validations','agent_revisions','agent_publication_commands')`).Scan(&foreign)
	if err != nil {
		return err
	}
	if foreign != 0 {
		return errors.New("foreign schema objects")
	}
	var metadata bool
	if err = d.conn.QueryRowContext(d.ctx, "SELECT to_regclass('public.schema_migrations') IS NOT NULL").Scan(&metadata); err != nil {
		return err
	}
	if adopting && metadata {
		return errors.New("baseline requires absent metadata")
	}
	if metadata {
		return d.matches("metadata.json", []string{"schema_migrations"})
	}

	return nil
}

// matches compares normalized catalog facts without reading or logging stored row values.
func (d *driver) matches(asset string, tables []string) error {
	q, err := catalogAssets.ReadFile("catalog.sql")
	if err != nil {
		return err
	}
	expected, err := catalogAssets.ReadFile(asset)
	if err != nil {
		return err
	}
	var actual string
	if err = d.conn.QueryRowContext(d.ctx, string(q), tables).Scan(&actual); err != nil {
		return err
	}
	var a, b any
	if err = json.Unmarshal([]byte(actual), &a); err != nil {
		return err
	}
	if err = json.Unmarshal(expected, &b); err != nil {
		return err
	}
	if !reflect.DeepEqual(a, b) {
		return errors.New("legacy schema drift")
	}
	return nil
}

// baseline adopts only the observed Task-only or full three-table schema under the migration lock.
func (d *driver) baseline() (int, bool, error) {
	if err := d.matches("tasks.json", []string{"tasks"}); err != nil {
		return 0, false, err
	}
	var count int
	if err := d.conn.QueryRowContext(d.ctx, `SELECT count(*) FROM information_schema.tables WHERE table_schema='public' AND table_name IN ('administrative_accounts','administrative_sessions')`).Scan(&count); err != nil {
		return 0, false, err
	}
	v := 1
	if count == 2 {
		if err := d.matches("administrative.json", []string{"administrative_accounts", "administrative_sessions"}); err != nil {
			return 0, false, err
		}
		v = 2
	} else if count != 0 {
		return 0, false, errors.New("partial administrative schema")
	}
	tx, err := d.conn.BeginTx(d.ctx, nil)
	if err != nil {
		return 0, false, err
	}
	defer func() { _ = tx.Rollback() }()
	if _, err = tx.ExecContext(d.ctx, metadataDDL); err != nil {
		return 0, false, err
	}
	if _, err = tx.ExecContext(d.ctx, "INSERT INTO public.schema_migrations (version, dirty) VALUES ($1, false)", v); err != nil {
		return 0, false, err
	}
	if err = tx.Commit(); err != nil {
		return 0, false, err
	}
	return v, false, nil
}
