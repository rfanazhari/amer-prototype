package schema_test

import (
	"context"
	"testing"

	"github.com/rfanazhari/amer-prototype/infrastructure/database/schema"
	"github.com/stretchr/testify/require"
)

// TestInterviewIntegrationMigrationPostgres proves populated upgrade, original legacy retention, constraints and disposable down.
func TestInterviewIntegrationMigrationPostgres(t *testing.T) {
	db, runner := openMigrationDatabase(t)
	ctx := context.Background()
	t.Run("fresh install creates the release", func(t *testing.T) {
		resetSchema(t, db)
		migrateTo(t, runner, 11)
		migrateTo(t, runner, 12)
		require.NoError(t, schema.RequireCompatible(ctx, db))
	})
	t.Run("populated ten upgrade and disposable down preserve pins and original rows", func(t *testing.T) {
		resetSchema(t, db)
		migrateTo(t, runner, 7)
		seedVersionSevenPin(t, db)
		migrateTo(t, runner, 10)
		_, err := db.ExecContext(ctx, `INSERT INTO interview_progress(execution_id,follow_up_limit,current_stage,stage_activated_at,committed_version,created_at,updated_at) VALUES ($1,2,'background',now(),1,now(),now());`, fixturePin)
		require.NoError(t, err)
		_, err = db.ExecContext(ctx, `INSERT INTO interview_session_commands(id,execution_id,idempotency_key,intent,attempt_id,created_at) VALUES ('77777777-7777-4777-8777-777777777771',$1,'legacy-key','activate','77777777-7777-4777-8777-777777777772',now())`, fixturePin)
		require.NoError(t, err)
		migrateTo(t, runner, 11)
		require.Equal(t, 1, countRows(t, db, "interview_executions"))
		require.Equal(t, 1, countRows(t, db, "interview_progress"))
		var state string
		var noResult bool
		require.NoError(t, db.QueryRowContext(ctx, `SELECT state, session_id IS NULL AND answer_sdp IS NULL AND execution_version IS NULL FROM interview_session_commands WHERE idempotency_key = 'legacy-key'`).Scan(&state, &noResult))
		require.Equal(t, "legacy", state)
		require.True(t, noResult)
		// Pending must own a real session; a fabricated success is refused.
		_, err = db.ExecContext(ctx, `INSERT INTO interview_session_commands(id,execution_id,idempotency_key,intent,attempt_id,state,created_at,updated_at) VALUES ('77777777-7777-4777-8777-777777777773',$1,'bad-pending','activate','77777777-7777-4777-8777-777777777772','pending',now(),now())`, fixturePin)
		require.Error(t, err)
		_, err = db.ExecContext(ctx, `UPDATE interview_session_commands SET state = 'succeeded', answer_sdp = 'fabricated', execution_version = 2 WHERE idempotency_key = 'legacy-key'`)
		require.Error(t, err)
		version, dirty, err := runner.Execute(ctx, schema.Options{Action: "down", Steps: 1, Disposable: true})
		require.NoError(t, err)
		require.Equal(t, 10, version)
		require.False(t, dirty)
		require.Equal(t, 1, countRows(t, db, "interview_session_commands"))
		require.Equal(t, 1, countRows(t, db, "interview_progress"))
		require.Equal(t, 1, countRows(t, db, "interview_executions"))
		migrateTo(t, runner, 11)
		migrateTo(t, runner, 12)
		require.NoError(t, schema.RequireCompatible(ctx, db))
	})
	t.Run("baseline recognizes integration columns", func(t *testing.T) {
		resetSchema(t, db)
		migrateTo(t, runner, 11)
		_, err := db.ExecContext(ctx, `DROP TABLE schema_migrations`)
		require.NoError(t, err)
		version, _, err := runner.Execute(ctx, schema.Options{Action: "baseline"})
		require.NoError(t, err)
		require.Equal(t, 11, version)
	})
}
