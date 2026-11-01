package schema_test

import (
	"context"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/rfanazhari/amer-prototype/infrastructure/database/schema"
	"github.com/stretchr/testify/require"
)

// TestStatusReadsOnly proves status observes metadata without creating it.
func TestStatusReadsOnly(t *testing.T) {
	db, m, e := sqlmock.New()
	require.NoError(t, e)
	defer func() { _ = db.Close() }()
	m.ExpectQuery("SELECT version, dirty FROM public.schema_migrations").WillReturnRows(sqlmock.NewRows([]string{"version", "dirty"}).AddRow(3, false))
	v, d, e := (schema.Runner{DB: db}).Execute(context.Background(), schema.Options{Action: "status"})
	require.NoError(t, e)
	require.Equal(t, 3, v)
	require.False(t, d)
	require.NoError(t, m.ExpectationsWereMet())
}

// TestOptionsRefuseUnsafeOperations proves invalid options fail before SQL.
func TestOptionsRefuseUnsafeOperations(t *testing.T) {
	// optionCase describes a forbidden or pre-SQL command.
	type optionCase struct {
		name     string
		options  schema.Options
		wantErr  bool
		contains string
	}
	for _, tt := range []optionCase{
		{name: "up requires target", options: schema.Options{Action: "up"}, wantErr: true},
		{name: "up accepts configuration target", options: schema.Options{Action: "up", Version: 3}, wantErr: true, contains: "database required"},
		{name: "up refuses unpublished target", options: schema.Options{Action: "up", Version: 4}, wantErr: true},
		{name: "down requires confirmation", options: schema.Options{Action: "down", Steps: 1}, wantErr: true},
		{name: "no force", options: schema.Options{Action: "force", Version: 2}, wantErr: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			_, _, e := (schema.Runner{}).Execute(context.Background(), tt.options)
			require.Error(t, e)
			if tt.contains != "" {
				require.ErrorContains(t, e, tt.contains)
			}
		})
	}
}
