package schema_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/rfanazhari/amer-prototype/infrastructure/database/schema"
	"github.com/stretchr/testify/require"
)

// artifactExpectation names a required version-3 or reallocated SQL fragment.
type artifactExpectation struct {
	name     string
	path     string
	contains []string
}

// TestAgentConfigurationArtifacts proves version 3 SQL, ownership constraints and unpublished file reallocation.
func TestAgentConfigurationArtifacts(t *testing.T) {
	root := filepath.Join("..", "..", "..")
	migrations := filepath.Join(root, "migrations", "pgsql")
	tests := []artifactExpectation{
		{
			name: "configuration up",
			path: filepath.Join(migrations, "000003_agent_configuration.up.sql"),
			contains: []string{
				"CREATE TABLE public.agent_instances",
				"CREATE TABLE public.agent_drafts",
				"CREATE TABLE public.agent_validations",
				"CREATE TABLE public.agent_revisions",
				"CREATE TABLE public.agent_publication_commands",
				"UNIQUE (instance_id, revision_number)",
				"UNIQUE (scope, command_kind, request_key)",
				"REFERENCES public.administrative_accounts(id) ON UPDATE RESTRICT ON DELETE RESTRICT",
				"FOREIGN KEY (id, current_draft_version) REFERENCES public.agent_drafts(instance_id, version)",
				"FOREIGN KEY (id, active_revision_id) REFERENCES public.agent_revisions(instance_id, id)",
				"FOREIGN KEY (instance_id, validation_id) REFERENCES public.agent_validations(instance_id, id)",
				"zero/null",
			},
		},
		{
			name: "configuration down",
			path: filepath.Join(migrations, "000003_agent_configuration.down.sql"),
			contains: []string{
				"DROP TABLE IF EXISTS public.agent_publication_commands",
				"DROP TABLE IF EXISTS public.agent_revisions",
				"DROP TABLE IF EXISTS public.agent_validations",
				"DROP TABLE IF EXISTS public.agent_drafts",
				"DROP TABLE IF EXISTS public.agent_instances",
			},
		},
		{
			name:     "reallocated candidates",
			path:     filepath.Join(migrations, "000004_candidates.up.sql"),
			contains: []string{"CREATE TABLE candidates"},
		},
		{
			name:     "reallocated assignment access",
			path:     filepath.Join(migrations, "000005_assignment_access.up.sql"),
			contains: []string{"CREATE TABLE interview_invitations"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, err := os.ReadFile(tt.path)
			require.NoError(t, err)
			for _, snippet := range tt.contains {
				require.Contains(t, string(body), snippet)
			}
		})
	}
	_, err := os.Stat(filepath.Join(migrations, "000003_candidates.up.sql"))
	require.ErrorIs(t, err, os.ErrNotExist)
	_, err = os.Stat(filepath.Join(migrations, "000004_assignment_access.up.sql"))
	require.ErrorIs(t, err, os.ErrNotExist)
	manifest, err := os.ReadFile(filepath.Join("checksums.json"))
	require.NoError(t, err)
	var sums map[string]string
	require.NoError(t, json.Unmarshal(manifest, &sums))
	require.Contains(t, sums, "000003_agent_configuration.up.sql")
	require.Contains(t, sums, "000003_agent_configuration.down.sql")
	require.NotContains(t, sums, "000004_candidates.up.sql")
	require.NotContains(t, sums, "000005_assignment_access.up.sql")
}

// TestAgentConfigurationVersionUpgrade proves fresh migrate-to-3 and upgrade from retained version 2 rows.
func TestAgentConfigurationVersionUpgrade(t *testing.T) {
	target := os.Getenv("TEST_MIGRATION_DATABASE_URL")
	if target == "" {
		t.Skip("TEST_MIGRATION_DATABASE_URL absent; configuration migration proof skipped")
	}
	db, err := sql.Open("pgx", target)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, db.Close()) })
	directory, err := filepath.Abs("../../../migrations/pgsql")
	require.NoError(t, err)
	runner := schema.Runner{DB: db, Directory: directory}
	// upgradeCase independently arranges a starting catalog before version 3.
	type upgradeCase struct {
		name    string
		prepare func(*testing.T, *sql.DB, schema.Runner)
	}
	tests := []upgradeCase{
		{name: "fresh migrate to 3", prepare: func(t *testing.T, db *sql.DB, _ schema.Runner) {
			t.Helper()
			_, err := db.ExecContext(context.Background(), "DROP SCHEMA public CASCADE; CREATE SCHEMA public;")
			require.NoError(t, err)
		}},
		{name: "upgrade from version 2 with retained rows", prepare: func(t *testing.T, db *sql.DB, runner schema.Runner) {
			t.Helper()
			_, err := db.ExecContext(context.Background(), "DROP SCHEMA public CASCADE; CREATE SCHEMA public;")
			require.NoError(t, err)
			version, dirty, err := runner.Execute(context.Background(), schema.Options{Action: "up", Version: 2})
			require.NoError(t, err)
			require.Equal(t, 2, version)
			require.False(t, dirty)
			_, err = db.ExecContext(context.Background(), `INSERT INTO tasks(id,title,status,created_at,updated_at) VALUES
('00000000-0000-0000-0000-000000000001','retained task','todo','2026-01-01Z','2026-01-01Z');
INSERT INTO administrative_accounts(id,name,email,password_hash,role,status,created_at,updated_at) VALUES
('00000000-0000-0000-0000-000000000002','Synthetic Admin','synthetic-admin@example.com','synthetic-hash','ADMIN','ACTIVE','2026-01-01Z','2026-01-01Z');`)
			require.NoError(t, err)
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.prepare(t, db, runner)
			beforeTasks, beforeAccounts := 0, 0
			if tt.name != "fresh migrate to 3" {
				require.NoError(t, db.QueryRowContext(context.Background(), "SELECT count(*) FROM tasks").Scan(&beforeTasks))
				require.NoError(t, db.QueryRowContext(context.Background(), "SELECT count(*) FROM administrative_accounts").Scan(&beforeAccounts))
			}
			version, dirty, err := runner.Execute(context.Background(), schema.Options{Action: "up", Version: 3})
			require.NoError(t, err)
			require.Equal(t, 3, version)
			require.False(t, dirty)
			require.NoError(t, schema.RequireCompatible(context.Background(), db))
			for _, table := range []string{"agent_instances", "agent_drafts", "agent_validations", "agent_revisions", "agent_publication_commands"} {
				var present bool
				require.NoError(t, db.QueryRowContext(context.Background(), "SELECT to_regclass('public.'||$1) IS NOT NULL", table).Scan(&present))
				require.True(t, present, table)
			}
			if tt.name != "fresh migrate to 3" {
				var tasks, accounts int
				require.NoError(t, db.QueryRowContext(context.Background(), "SELECT count(*) FROM tasks").Scan(&tasks))
				require.NoError(t, db.QueryRowContext(context.Background(), "SELECT count(*) FROM administrative_accounts").Scan(&accounts))
				require.Equal(t, beforeTasks, tasks)
				require.Equal(t, beforeAccounts, accounts)
			}
		})
	}
}
