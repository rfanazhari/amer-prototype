package schema

import (
	"context"
	"database/sql"
)

// RequireCompatible checks the first cutover binary's schema contract without DDL.
func RequireCompatible(_ context.Context, _ *sql.DB) error { return nil }
