package service_test

import (
	"testing"

	"github.com/rfanazhari/amer-prototype/application/service"
	"github.com/rfanazhari/amer-prototype/domain/enum"
	"github.com/stretchr/testify/assert"
)

// TestAllowsInterviewReport proves interview report access is HR-only and independent of configuration access.
func TestAllowsInterviewReport(t *testing.T) {
	tests := []struct {
		name     string
		role     enum.Role
		expected bool
	}{
		{name: "hr is allowed", role: enum.HR, expected: true},
		{name: "admin is denied", role: enum.Admin, expected: false},
		{name: "empty role is denied", role: enum.Role(""), expected: false},
		{name: "unknown role is denied", role: enum.Role("CANDIDATE"), expected: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, service.AllowsInterviewReport(tt.role))
		})
	}
}

// TestKnownInterviewReportOperation proves only the report operations are recognized.
func TestKnownInterviewReportOperation(t *testing.T) {
	tests := []struct {
		name      string
		operation string
		expected  bool
	}{
		{name: "read", operation: service.InterviewReportRead, expected: true},
		{name: "regenerate", operation: service.InterviewReportRegenerate, expected: true},
		{name: "configuration create is not a report operation", operation: service.AgentConfigurationCreate, expected: false},
		{name: "configuration publish is not a report operation", operation: service.AgentConfigurationPublish, expected: false},
		{name: "empty", operation: "", expected: false},
		{name: "unknown", operation: "delete", expected: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, service.KnownInterviewReportOperation(tt.operation))
		})
	}
}

// TestInterviewReportPolicyIsSeparateFromConfiguration proves the two predicates do not share a decision list.
func TestInterviewReportPolicyIsSeparateFromConfiguration(t *testing.T) {
	assert.False(t, service.KnownAgentConfigurationOperation(service.InterviewReportRegenerate))
	assert.True(t, service.KnownAgentConfigurationOperation(service.AgentConfigurationRead))
	assert.True(t, service.KnownInterviewReportOperation(service.InterviewReportRead))
}
