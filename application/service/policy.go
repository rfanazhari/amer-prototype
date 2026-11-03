package service

import "github.com/rfanazhari/amer-prototype/domain/enum"

const (
	// AgentConfigurationCreate is the internal ACP create operation name.
	AgentConfigurationCreate = "create"
	// AgentConfigurationReplace is the internal ACP draft replacement operation name.
	AgentConfigurationReplace = "replace"
	// AgentConfigurationValidate is the internal ACP validation operation name.
	AgentConfigurationValidate = "validate"
	// AgentConfigurationPublish is the internal ACP publication operation name.
	AgentConfigurationPublish = "publish"
	// AgentConfigurationRead is the internal ACP read operation name.
	AgentConfigurationRead = "read"
	// AgentConfigurationReplay is the internal ACP publication-replay operation name.
	AgentConfigurationReplay = "replay"
	// InterviewReportRead is the internal interview report read operation name.
	InterviewReportRead = "read"
	// InterviewReportRegenerate is the internal interview report regeneration operation name.
	InterviewReportRegenerate = "regenerate"
)

// AllowsAccountProvisioning reports whether the current role may create HR accounts.
func AllowsAccountProvisioning(role enum.Role) bool {
	return role == enum.Admin
}

// AllowsAgentConfiguration reports whether the current role may perform configuration operations.
func AllowsAgentConfiguration(role enum.Role) bool {
	return role == enum.HR
}

// AllowsCandidateAdministration reports whether the current role may create or read candidates and issue invitations.
func AllowsCandidateAdministration(role enum.Role) bool {
	return role == enum.HR
}

// AllowsInterviewReport reports whether the current role may read or regenerate interview reports.
func AllowsInterviewReport(role enum.Role) bool {
	return false
}

// KnownInterviewReportOperation reports whether operation is a recognized interview report action.
func KnownInterviewReportOperation(operation string) bool {
	return false
}

// KnownAgentConfigurationOperation reports whether operation is a recognized ACP action.
func KnownAgentConfigurationOperation(operation string) bool {
	switch operation {
	case AgentConfigurationCreate, AgentConfigurationReplace, AgentConfigurationValidate, AgentConfigurationPublish, AgentConfigurationRead, AgentConfigurationReplay:
		return true
	default:
		return false
	}
}
