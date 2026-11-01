package schema_test

import (
	"context"
	"github.com/DATA-DOG/go-sqlmock"
	"github.com/rfanazhari/amer-prototype/infrastructure/database/schema"
	"github.com/stretchr/testify/require"
	"testing"
)

// TestStatusReadsOnly proves status observes metadata without creating it.
func TestStatusReadsOnly(t *testing.T) {
	db, m, e := sqlmock.New()
	require.NoError(t, e)
	defer db.Close()
	m.ExpectQuery("SELECT version, dirty FROM public.schema_migrations").WillReturnRows(sqlmock.NewRows([]string{"version", "dirty"}).AddRow(2, false))
	v, d, e := (schema.Runner{DB: db}).Execute(context.Background(), schema.Options{Action: "status"})
	require.NoError(t, e)
	require.Equal(t, 2, v)
	require.False(t, d)
	require.NoError(t, m.ExpectationsWereMet())
}

// TestOptionsRefuseUnsafeOperations proves invalid options fail before SQL.
func TestOptionsRefuseUnsafeOperations(t *testing.T) {
	// optionCase describes a forbidden command.
	type optionCase struct {
		name    string
		options schema.Options
		wantErr bool
	}
	for _, tt := range []optionCase{{"up requires target", schema.Options{Action: "up"}, true}, {"down requires confirmation", schema.Options{Action: "down", Steps: 1}, true}, {"no force", schema.Options{Action: "force", Version: 2}, true}} {
		t.Run(tt.name, func(t *testing.T) {
			_, _, e := (schema.Runner{}).Execute(context.Background(), tt.options)
			require.Error(t, e)
		})
	}
}
