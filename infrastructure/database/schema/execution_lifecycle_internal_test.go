package schema

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestOwnedTablesIncludeExecutionLifecycle proves the foreign-schema guard knows the version 8 tables.
func TestOwnedTablesIncludeExecutionLifecycle(t *testing.T) {
	for _, table := range []string{
		"agent_executions",
		"agent_execution_commands",
		"agent_execution_operations",
		"agent_execution_attempts",
		"agent_execution_transitions",
	} {
		require.Contains(t, ownedPublicTables, table)
		require.True(t, strings.Contains(ownedTableList(), "'"+table+"'"), table)
	}
	require.Contains(t, ownedPublicTables, "interview_executions")
}
