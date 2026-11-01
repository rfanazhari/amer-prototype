package contract_test

import (
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
