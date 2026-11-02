package schema

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestOwnedTablesIncludeInterviewRuntime proves the foreign-schema guard knows the version 10 tables.
func TestOwnedTablesIncludeInterviewRuntime(t *testing.T) {
	for _, table := range []string{
		"interview_progress",
		"interview_coverage_items",
		"interview_technical_losses",
		"interview_voice_sessions",
		"interview_transcript_entries",
		"interview_checkpoints",
		"interview_handoffs",
		"interview_session_commands",
	} {
		require.Contains(t, ownedPublicTables, table)
		require.True(t, strings.Contains(ownedTableList(), "'"+table+"'"), table)
	}
	require.Contains(t, ownedPublicTables, "interview_executions")
}
