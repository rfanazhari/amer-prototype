package schema

import (
	"embed"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
)

// catalogAssets freezes independently observed legacy catalog facts.
//
//go:embed catalog.sql tasks.json administrative.json metadata.json checksums.json
var catalogAssets embed.FS

// preflight rejects foreign schemas/metadata before any tool-owned DDL.
func (d *driver) preflight(adopting bool) error {
	var foreign int
	err := d.conn.QueryRowContext(d.ctx, `SELECT count(*) FROM information_schema.tables WHERE table_schema='public' AND table_name NOT IN (`+ownedTableList()+`)`).Scan(&foreign)
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

// baseline adopts Task-only, full three-table, or already-installed configuration catalogs under the migration lock.
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
	var configuration int
	if err := d.conn.QueryRowContext(d.ctx, `SELECT count(*) FROM information_schema.tables WHERE table_schema='public' AND table_name IN ('agent_instances','agent_drafts','agent_validations','agent_revisions','agent_publication_commands')`).Scan(&configuration); err != nil {
		return 0, false, err
	}
	if configuration == 5 {
		if v != 2 {
			return 0, false, errors.New("configuration schema requires administrative tables")
		}
		v = 3
	} else if configuration != 0 {
		return 0, false, errors.New("partial configuration schema")
	}
	if v == 3 {
		next, err := d.extendBaseline(v)
		if err != nil {
			return 0, false, err
		}
		v = next
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

// extendBaseline stamps versions 4–8 when later owned tables are already present.
func (d *driver) extendBaseline(v int) (int, error) {
	candidates, err := d.countNamedTables("candidates")
	if err != nil {
		return 0, err
	}
	if candidates == 1 {
		v = 4
	} else if candidates != 0 {
		return 0, errors.New("partial candidate schema")
	}
	assignments, err := d.countNamedTables("interview_assignments")
	if err != nil {
		return 0, err
	}
	if assignments == 1 {
		if v != 4 {
			return 0, errors.New("assignment schema requires candidates")
		}
		v = 5
	} else if assignments != 0 {
		return 0, errors.New("partial assignment schema")
	}
	access, err := d.countNamedTables("interview_invitations", "candidate_sessions")
	if err != nil {
		return 0, err
	}
	if access == 2 {
		if v != 5 {
			return 0, errors.New("assignment access schema requires interview_assignments")
		}
		v = 6
	} else if access != 0 {
		return 0, errors.New("partial assignment access schema")
	}
	executions, err := d.countNamedTables("interview_executions")
	if err != nil {
		return 0, err
	}
	if executions == 1 {
		if v != 6 {
			return 0, errors.New("execution schema requires assignment access")
		}
		v = 7
	} else if executions != 0 {
		return 0, errors.New("partial execution schema")
	}
	lifecycle, err := d.countNamedTables("agent_executions", "agent_execution_commands", "agent_execution_operations", "agent_execution_attempts", "agent_execution_transitions")
	if err != nil {
		return 0, err
	}
	if lifecycle == 5 {
		if v != 7 {
			return 0, errors.New("execution lifecycle schema requires execution pins")
		}
		v = 8
	} else if lifecycle != 0 {
		return 0, errors.New("partial execution lifecycle schema")
	}
	return v, nil
}

// countNamedTables counts matching public tables from the supplied names.
func (d *driver) countNamedTables(names ...string) (int, error) {
	quoted := make([]string, len(names))
	for i, name := range names {
		quoted[i] = "'" + name + "'"
	}
	var count int
	err := d.conn.QueryRowContext(d.ctx, `SELECT count(*) FROM information_schema.tables WHERE table_schema='public' AND table_name IN (`+strings.Join(quoted, ",")+`)`).Scan(&count)
	return count, err
}
