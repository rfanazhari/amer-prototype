package schema_test

import (
	"context"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/rfanazhari/amer-prototype/infrastructure/database/schema"
	"github.com/stretchr/testify/require"
)

// deploymentCase names one stored-identity deployment compatibility expectation.
type deploymentCase struct {
	name    string
	rows    [][]string
	wantErr string
}

// TestRequireAgentConfigurationDeploymentCompatible covers unsupported behavior rejection and rollback rehearsal.
func TestRequireAgentConfigurationDeploymentCompatible(t *testing.T) {
	ctx := context.Background()
	tests := []deploymentCase{
		{
			name: "empty configuration tables allow startup",
			rows: nil,
		},
		{
			name: "rollback rehearsal retains readable version-one behavior",
			rows: [][]string{
				{"hr-interviewer", "1", "1"},
				{"document-analyzer", "1", "1"},
			},
		},
		{
			name: "unsupported stored behavior rejects deployment explicitly",
			rows: [][]string{{"hr-interviewer", "1", "2"}},
			wantErr: "incompatible agent configuration deployment; unsupported stored identities: hr-interviewer@1 behavior 2",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			require.NoError(t, err)
			t.Cleanup(func() { _ = db.Close() })
			rows := sqlmock.NewRows([]string{"template_id", "template_version", "behavior_version"})
			for _, item := range tt.rows {
				rows.AddRow(item[0], item[1], item[2])
			}
			mock.ExpectQuery("SELECT template_id, template_version, behavior_version").
				WillReturnRows(rows)
			err = schema.RequireAgentConfigurationDeploymentCompatible(ctx, db)
			if tt.wantErr == "" {
				require.NoError(t, err)
				return
			}
			require.EqualError(t, err, tt.wantErr)
		})
	}
}
