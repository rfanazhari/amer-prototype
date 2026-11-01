package schema

import "strings"

// ReleaseVersion is the only clean schema_migrations.version accepted at application startup.
const ReleaseVersion = 9

// ownedPublicTables lists tool-owned public tables that are not foreign schema.
var ownedPublicTables = []string{
	"schema_migrations",
	"tasks",
	"administrative_accounts",
	"administrative_sessions",
	"agent_instances",
	"agent_drafts",
	"agent_validations",
	"agent_revisions",
	"agent_publication_commands",
	"candidates",
	"interview_assignments",
	"interview_invitations",
	"candidate_sessions",
	"interview_executions",
	"agent_executions",
	"agent_execution_commands",
	"agent_execution_operations",
	"agent_execution_attempts",
	"agent_execution_transitions",
	"interview_consents",
	"interview_controllers",
}

// RuntimeGrantTables are the business tables that receive application CRUD.
var RuntimeGrantTables = []string{
	"tasks",
	"administrative_accounts",
	"administrative_sessions",
	"agent_instances",
	"agent_drafts",
	"agent_validations",
	"agent_revisions",
	"agent_publication_commands",
	"candidates",
	"interview_assignments",
	"interview_invitations",
	"candidate_sessions",
	"interview_executions",
	"agent_executions",
	"agent_execution_commands",
	"agent_execution_operations",
	"agent_execution_attempts",
	"agent_execution_transitions",
	"interview_consents",
	"interview_controllers",
}

// ownedTableList formats owned public table names for an SQL IN list.
func ownedTableList() string {
	quoted := make([]string, len(ownedPublicTables))
	for i, name := range ownedPublicTables {
		quoted[i] = "'" + name + "'"
	}
	return strings.Join(quoted, ",")
}
