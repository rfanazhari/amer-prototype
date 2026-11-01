package schema_test

import (
	"context"
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/rfanazhari/amer-prototype/infrastructure/database/schema"
	"github.com/stretchr/testify/require"
)

// compatibilityCase describes the metadata accepted by the configuration-aware binary.
type compatibilityCase struct {
	name     string
	rows     *sqlmock.Rows
	queryErr error
	wantErr  bool
}

// TestRequireCompatible proves read-only exact-version, cardinality and failure checks.
func TestRequireCompatible(t *testing.T) {
	tests := []compatibilityCase{
		{name: "clean release", rows: sqlmock.NewRows([]string{"version", "dirty"}).AddRow(schema.ReleaseVersion, false)},
		{name: "old", rows: sqlmock.NewRows([]string{"version", "dirty"}).AddRow(3, false), wantErr: true},
		{name: "future", rows: sqlmock.NewRows([]string{"version", "dirty"}).AddRow(schema.ReleaseVersion+1, false), wantErr: true},
		{name: "dirty", rows: sqlmock.NewRows([]string{"version", "dirty"}).AddRow(schema.ReleaseVersion, true), wantErr: true},
		{name: "empty", rows: sqlmock.NewRows([]string{"version", "dirty"}), wantErr: true},
		{name: "ambiguous", rows: sqlmock.NewRows([]string{"version", "dirty"}).AddRow(schema.ReleaseVersion, false).AddRow(schema.ReleaseVersion+1, false), wantErr: true},
		{name: "missing or unreadable", queryErr: errors.New("private database diagnostic"), wantErr: true},
		{name: "iteration failure", rows: sqlmock.NewRows([]string{"version", "dirty"}).AddRow(schema.ReleaseVersion, false).RowError(0, errors.New("private row diagnostic")), wantErr: true},
		{name: "malformed", rows: sqlmock.NewRows([]string{"version", "dirty"}).AddRow("invalid", false), wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			require.NoError(t, err)
			defer func() { _ = db.Close() }()
			expect := mock.ExpectQuery("SELECT version, dirty FROM public.schema_migrations")
			if tt.queryErr != nil {
				expect.WillReturnError(tt.queryErr)
			} else {
				expect.WillReturnRows(tt.rows)
			}
			err = schema.RequireCompatible(context.Background(), db)
			if tt.wantErr {
				require.Error(t, err)
				require.NotContains(t, err.Error(), "private")
			} else {
				require.NoError(t, err)
			}
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}
