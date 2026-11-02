package schema_test

import (
	"context"
	"database/sql"
	"github.com/stretchr/testify/require"
	"testing"
)

// seedInterviewRuntimeRollback populates every version-10 table, including coverage citations and session dependents.
func seedInterviewRuntimeRollback(t *testing.T, db *sql.DB) {
	t.Helper()
	statements := []string{
		`INSERT INTO interview_progress(execution_id,follow_up_limit,current_stage,stage_activated_at,committed_version,created_at,updated_at) VALUES ($1,2,'background','2026-01-02T03:04:05.678Z',1,'2026-01-02T03:04:05.678Z','2026-01-02T03:04:05.678Z')`,
		`INSERT INTO interview_voice_sessions(id,execution_id,stage,attempt_id,opened_at) VALUES ('10000000-0000-4000-8000-000000000001',$1,'background','10000000-0000-4000-8000-000000000002','2026-01-02T03:04:05.678Z')`,
		`INSERT INTO interview_transcript_entries(id,execution_id,session_id,attempt_id,speaker,ordinal,revision,evidence_state,body,committed_at) VALUES ('10000000-0000-4000-8000-000000000003',$1,'10000000-0000-4000-8000-000000000001','10000000-0000-4000-8000-000000000002','candidate',1,1,'final','synthetic rollback evidence','2026-01-02T03:04:05.678Z')`,
		`INSERT INTO interview_coverage_items(id,execution_id,question_id,stage,state,clarification_count,answer_entry_id,updated_at) VALUES ('10000000-0000-4000-8000-000000000004',$1,'background','background','answered',0,'10000000-0000-4000-8000-000000000003','2026-01-02T03:04:05.678Z')`,
		`INSERT INTO interview_technical_losses(id,execution_id,question_id,session_id,attempt_id,created_at) VALUES ('10000000-0000-4000-8000-000000000005',$1,'background','10000000-0000-4000-8000-000000000001','10000000-0000-4000-8000-000000000002','2026-01-02T03:04:05.678Z')`,
		`INSERT INTO interview_checkpoints(id,execution_id,session_id,attempt_id,through_ordinal,created_at) VALUES ('10000000-0000-4000-8000-000000000006',$1,'10000000-0000-4000-8000-000000000001','10000000-0000-4000-8000-000000000002',1,'2026-01-02T03:04:05.678Z')`,
		`INSERT INTO interview_handoffs(id,execution_id,source_stage,target_stage,revision,created_at) VALUES ('10000000-0000-4000-8000-000000000007',$1,'background','screening',1,'2026-01-02T03:04:05.678Z')`,
		`INSERT INTO interview_session_commands(id,execution_id,idempotency_key,intent,attempt_id,created_at) VALUES ('10000000-0000-4000-8000-000000000008',$1,'rollback-key','activate','10000000-0000-4000-8000-000000000002','2026-01-02T03:04:05.678Z')`,
	}
	for _, statement := range statements {
		_, err := db.ExecContext(context.Background(), statement, fixturePin)
		require.NoError(t, err)
	}
}
