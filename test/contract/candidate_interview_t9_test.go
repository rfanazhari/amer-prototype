package contract_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestOpenAPIPublishesCandidateExecutionRoutes requires T9 operations, statuses, strict bodies and candidate authority.
func TestOpenAPIPublishesCandidateExecutionRoutes(t *testing.T) {
	doc := loadContract(t)
	tests := []mountedInterviewOperation{
		{path: "/candidate/executions", method: "POST", statuses: []string{"200", "201", "400", "401", "403", "405", "409", "500", "503"}, security: true},
		{path: "/candidate/executions/{id}", method: "GET", statuses: []string{"200", "400", "401", "404", "405", "500", "503"}, security: true},
		{path: "/candidate/executions/{id}/takeover", method: "POST", statuses: []string{"200", "400", "401", "403", "404", "405", "409", "500", "503"}, security: true},
		{path: "/candidate/executions/{id}/cancel", method: "POST", statuses: []string{"200", "400", "401", "403", "404", "405", "409", "500", "503"}, security: true},
		{path: "/candidate/executions/{id}/complete", method: "POST", statuses: []string{"200", "400", "401", "403", "404", "405", "409", "500", "503"}, security: true},
	}
	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			item := doc.Paths.Find(tt.path)
			require.NotNil(t, item)
			op := item.GetOperation(tt.method)
			require.NotNil(t, op)
			require.NotNil(t, op.Security)
			require.Contains(t, (*op.Security)[0], "CandidateCookie")
			for _, status := range tt.statuses {
				require.NotNil(t, op.Responses.Value(status), status)
			}
			if tt.method == "POST" {
				require.NotNil(t, op.RequestBody)
				schema := op.RequestBody.Value.Content["application/json"].Schema.Value
				require.Equal(t, "false", schemaAdditional(schema))
			}
			if tt.path == "/candidate/executions" {
				require.NotNil(t, op.Parameters.GetByInAndName("header", "Idempotency-Key"))
				require.True(t, op.Parameters.GetByInAndName("header", "Idempotency-Key").Required)
			}
		})
	}
}

// TestOpenAPIPublishesCandidateFinish requires the finish action, its strict version body and the matching documentation.
func TestOpenAPIPublishesCandidateFinish(t *testing.T) {
	doc := loadContract(t)
	op := doc.Paths.Find("/candidate/executions/{id}/complete").GetOperation("POST")
	require.NotNil(t, op)
	require.Equal(t, "completeCandidateExecution", op.OperationID)
	schema := op.RequestBody.Value.Content["application/json"].Schema.Value
	require.Equal(t, []string{"expectedVersion"}, schema.Required)
	view := doc.Components.Schemas["CandidateInterviewView"].Value
	actions := view.Properties["actions"].Value.Items.Value.Enum
	require.Contains(t, actions, "complete")
	endpointDoc, err := os.ReadFile(filepath.Join("..", "..", "docs", "api", "candidate-interview.md"))
	require.NoError(t, err)
	require.Contains(t, string(endpointDoc), "POST `/candidate/executions/{id}/complete`")
	changelog, err := os.ReadFile(filepath.Join("..", "..", "docs", "api", "CHANGELOG.md"))
	require.NoError(t, err)
	require.Contains(t, string(changelog), "/candidate/executions/{id}/complete")
}
