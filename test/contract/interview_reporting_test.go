package contract_test

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// reportContractCase identifies a published HR handler contract and its success schema.
type reportContractCase struct {
	name     string
	path     string
	method   string
	statuses []string
	schema   string
}

// TestInterviewReportContract verifies the two HR contracts without claiming runtime registration.
func TestInterviewReportContract(t *testing.T) {
	doc := loadContract(t)
	tests := []reportContractCase{
		{name: "read", path: "/admin/interview-assignments/{id}/report", method: "GET", statuses: []string{"200", "400", "401", "403", "404", "405", "409", "500", "503"}, schema: "InterviewReportEnvelope"},
		{name: "regenerate", path: "/admin/interview-assignments/{id}/report-generations", method: "POST", statuses: []string{"200", "201", "400", "401", "403", "404", "405", "409", "500", "503"}, schema: "ReportGenerationEnvelope"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			item := doc.Paths.Find(tt.path)
			require.NotNil(t, item)
			op := item.GetOperation(tt.method)
			require.NotNil(t, op)
			require.Contains(t, op.Description, "not mounted in runtime until T15")
			require.NotNil(t, op.Security)
			require.Equal(t, []string{}, (*op.Security)[0]["AdminCookie"])
			for _, status := range tt.statuses {
				require.NotNil(t, op.Responses.Value(status), status)
			}
			require.Equal(t, "#/components/schemas/"+tt.schema, op.Responses.Value("200").Value.Content["application/json"].Schema.Ref)
			if tt.method == "POST" {
				require.True(t, op.RequestBody.Value.Required)
				body := op.RequestBody.Value.Content["application/json"].Schema.Value
				require.Empty(t, body.Properties)
				require.Equal(t, "false", schemaAdditional(body))
			}
		})
	}
}

// reportSchemaCase specifies exact property sets to exclude verdict and salary additions.
type reportSchemaCase struct {
	name       string
	properties []string
}

// TestInterviewReportSchemasAllowOnlyApprovedFields protects nullable and evidence-specific wire shapes.
func TestInterviewReportSchemasAllowOnlyApprovedFields(t *testing.T) {
	doc := loadContract(t)
	tests := []reportSchemaCase{
		{name: "InterviewReport", properties: []string{"assignmentId", "executionId", "generation", "report"}},
		{name: "ReportGeneration", properties: []string{"id", "number", "state", "failureReason", "createdAt"}},
		{name: "ReportAccount", properties: []string{"generationId", "observations", "gaps", "uncertainties"}},
		{name: "ReportObservation", properties: []string{"body", "evidenceEntryId", "evidenceRevision"}},
		{name: "ReportGap", properties: []string{"body", "questionId"}},
		{name: "ReportUncertainty", properties: []string{"body", "evidenceEntryId", "evidenceRevision"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := doc.Components.Schemas[tt.name].Value
			require.ElementsMatch(t, tt.properties, propertyNames(s))
			require.Equal(t, "false", schemaAdditional(s))
		})
	}
	require.True(t, doc.Components.Schemas["ReportAccount"].Value.Nullable)
	require.True(t, doc.Components.Schemas["ReportGeneration"].Value.Properties["failureReason"].Value.Nullable)
	require.True(t, doc.Components.Schemas["ReportGap"].Value.Properties["questionId"].Value.Nullable)
}
