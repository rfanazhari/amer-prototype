package schema

import "strings"

// ReleaseVersion is the only clean schema_migrations.version accepted at application startup.
const ReleaseVersion = 13

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
	"interview_progress",
	"interview_coverage_items",
	"interview_technical_losses",
	"interview_voice_sessions",
	"interview_transcript_entries",
	"interview_checkpoints",
	"interview_handoffs",
	"interview_session_commands",
	"interview_reports",
	"interview_report_generations",
	"interview_report_statements",
	"interview_report_commands",
	"interview_report_jobs",
	"interview_report_attempts",
	"outbox_messages",
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
	"interview_progress",
	"interview_coverage_items",
	"interview_technical_losses",
	"interview_voice_sessions",
	"interview_transcript_entries",
	"interview_checkpoints",
	"interview_handoffs",
	"interview_session_commands",
	"interview_reports",
	"interview_report_generations",
	"interview_report_statements",
	"interview_report_commands",
	"interview_report_jobs",
	"interview_report_attempts",
	"outbox_messages",
}

// ownedTableList formats owned public table names for an SQL IN list.
func ownedTableList() string {
	quoted := make([]string, len(ownedPublicTables))
	for i, name := range ownedPublicTables {
		quoted[i] = "'" + name + "'"
	}
	return strings.Join(quoted, ",")
}
