package schema

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

// RequireCompatible checks the configuration-aware binary's schema contract without DDL.
func RequireCompatible(ctx context.Context, db *sql.DB) error {
	if db == nil {
		return errors.New("schema check requires database")
	}
	rows, err := db.QueryContext(ctx, "SELECT version, dirty FROM public.schema_migrations")
	if err != nil {
		return fmt.Errorf("schema unavailable; run explicit migrations to clean version %d before startup", ReleaseVersion)
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
		if version != ReleaseVersion || dirty {
			return fmt.Errorf("incompatible schema; require clean version %d", ReleaseVersion)
		}
	}
	if rows.Err() != nil {
		return errors.New("schema metadata read failed")
	}
	if count != 1 {
		return fmt.Errorf("schema metadata must contain exactly one clean version %d", ReleaseVersion)
	}
	return nil
}
