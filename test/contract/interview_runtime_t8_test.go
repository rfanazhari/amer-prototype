package contract_test

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestOpenAPIPublishesSessionAttemptRoutes verifies the T8 routes, statuses, security and the answer-only response.
func TestOpenAPIPublishesSessionAttemptRoutes(t *testing.T) {
	doc := loadContract(t)
	tests := []mountedInterviewOperation{
		{path: "/candidate/executions/{id}/session-attempts", method: "POST", statuses: []string{"200", "201", "400", "401", "403", "404", "405", "409", "500", "503"}, security: true},
		{path: "/candidate/executions/{id}/session-attempts/{attempt_id}/ready", method: "POST", statuses: []string{"200", "400", "401", "403", "404", "405", "409", "500"}, security: true},
	}
	for _, tt := range tests {
		t.Run(tt.method+" "+tt.path, func(t *testing.T) {
			item := doc.Paths.Find(tt.path)
			require.NotNil(t, item)
			op := item.GetOperation(tt.method)
			require.NotNil(t, op)
			for _, status := range tt.statuses {
				require.NotNil(t, op.Responses.Value(status), status)
			}
			require.NotNil(t, op.Security)
			_, ok := (*op.Security)[0]["CandidateCookie"]
			require.True(t, ok)
		})
	}
}

// TestOpenAPISessionAttemptSchemasAreAllowLists verifies the request and response members and that nothing provider-internal is exposed.
func TestOpenAPISessionAttemptSchemasAreAllowLists(t *testing.T) {
	doc := loadContract(t)
	request := doc.Components.Schemas["CandidateSessionAttemptRequest"]
	require.NotNil(t, request)
	require.ElementsMatch(t, []string{"sdp", "intent", "expectedVersion", "controllerId", "idempotencyKey"}, propertyNames(request.Value))
	require.Equal(t, "false", schemaAdditional(request.Value))
	answer := doc.Components.Schemas["CandidateSessionAttemptEnvelope"].Value.Properties["data"].Value
	require.ElementsMatch(t, []string{"attemptId", "sessionId", "sdp", "executionVersion", "sessionCondition"}, propertyNames(answer))
	ready := doc.Components.Schemas["CandidateSessionReadyEnvelope"].Value.Properties["data"].Value
	require.ElementsMatch(t, []string{"attemptId", "sessionId", "executionVersion", "sessionCondition"}, propertyNames(ready))
}
