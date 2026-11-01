package schema

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestOwnedTablesIncludeCandidateInterview proves the foreign-schema guard knows the version 9 tables.
func TestOwnedTablesIncludeCandidateInterview(t *testing.T) {
	for _, table := range []string{"interview_consents", "interview_controllers"} {
		require.Contains(t, ownedPublicTables, table)
		require.True(t, strings.Contains(ownedTableList(), "'"+table+"'"), table)
	}
	require.Contains(t, ownedPublicTables, "interview_executions")
}
