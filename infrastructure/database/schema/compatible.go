package schema

import (
	"context"
	"database/sql"
	"errors"
)

// RequireCompatible checks the first cutover binary's schema contract without DDL.
func RequireCompatible(ctx context.Context, db *sql.DB) error {
	if db == nil {
		return errors.New("schema check requires database")
	}
	rows, err := db.QueryContext(ctx, "SELECT version, dirty FROM public.schema_migrations")
	if err != nil {
		return errors.New("schema unavailable; run explicit migrations to clean version 2 before startup")
	}
	defer func() { _ = rows.Close() }()
	count := 0
	for rows.Next() {
		var version int
		var dirty bool
		if err = rows.Scan(&version, &dirty); err != nil {
			return errors.New("unreadable schema metadata")
		}
		count++
		if version != 2 || dirty {
			return errors.New("incompatible schema; require clean version 2")
		}
	}
	if rows.Err() != nil {
		return errors.New("schema metadata read failed")
	}
	if count != 1 {
		return errors.New("schema metadata must contain exactly one clean version 2")
	}
	return nil
}
