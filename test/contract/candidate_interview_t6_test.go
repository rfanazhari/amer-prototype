package contract_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/stretchr/testify/require"
)

// mountedInterviewOperation names one T6 operation the OpenAPI document must publish.
type mountedInterviewOperation struct {
	path     string
	method   string
	statuses []string
	security bool
}

// loadContract loads the published OpenAPI document.
func loadContract(t *testing.T) *openapi3.T {
	t.Helper()
	doc, err := openapi3.NewLoader().LoadFromFile("../../docs/api/openapi.yaml")
	require.NoError(t, err)
	return doc
}

// TestOpenAPIPublishesMountedCandidateInterviewRoutes verifies the T6 routes, their statuses and their security.
func TestOpenAPIPublishesMountedCandidateInterviewRoutes(t *testing.T) {
	doc := loadContract(t)
	tests := []mountedInterviewOperation{
		{path: "/candidate/interview", method: "GET", statuses: []string{"200", "400", "401", "405", "500"}, security: true},
		{path: "/candidate/consents", method: "POST", statuses: []string{"200", "201", "400", "401", "403", "405", "500", "503"}, security: true},
		{path: "/candidate/interview/page", method: "GET", statuses: []string{"200", "405"}},
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
			if tt.security {
				require.NotNil(t, op.Security)
				require.Len(t, *op.Security, 1)
				_, ok := (*op.Security)[0]["CandidateCookie"]
				require.True(t, ok)
			}
		})
	}
}

// TestOpenAPIKeepsUnmountedInterviewRoutesAbsent verifies sibling runtime and reporting routes are not published.
func TestOpenAPIKeepsUnmountedInterviewRoutesAbsent(t *testing.T) {
	doc := loadContract(t)
	for path := range doc.Paths.Map() {
		require.NotContains(t, path, "/report")
		require.NotContains(t, path, "session-attempts")
		require.NotContains(t, path, "/ready")
	}
	require.NotNil(t, doc.Paths.Find("/candidate/session"))
	require.NotNil(t, doc.Paths.Find("/candidate/access/exchange"))
}

// TestOpenAPIInterviewViewIsAnAllowList verifies the view schema has no controller id, salary or report property.
func TestOpenAPIInterviewViewIsAnAllowList(t *testing.T) {
	doc := loadContract(t)
	schema := doc.Components.Schemas["CandidateInterviewView"]
	require.NotNil(t, schema)
	require.Equal(t, "false", schemaAdditional(schema.Value))
	require.ElementsMatch(t, []string{"notice", "consent", "execution", "actions", "progress"}, propertyNames(schema.Value))
	execution := schema.Value.Properties["execution"].Value
	require.NotNil(t, execution)
	require.ElementsMatch(t, []string{"id", "state", "version", "hardDeadline", "isCurrentController", "controllerVersion"}, propertyNames(execution))
	for _, name := range append(propertyNames(schema.Value), propertyNames(execution)...) {
		lower := strings.ToLower(name)
		for _, forbidden := range []string{"salary", "report", "controllerid", "transcript"} {
			require.NotContains(t, lower, forbidden)
		}
	}
	request := doc.Components.Schemas["CandidateConsentRequest"]
	require.NotNil(t, request)
	require.Equal(t, "false", schemaAdditional(request.Value))
	require.ElementsMatch(t, []string{"noticeVersion", "accepted"}, propertyNames(request.Value))
}

// TestOpenAPIPublishesCandidateInterviewErrorCodes verifies the CIN_ codes appear in the published contract.
func TestOpenAPIPublishesCandidateInterviewErrorCodes(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "docs", "api", "openapi.yaml"))
	require.NoError(t, err)
	for _, code := range []string{"CIN_FORBIDDEN", "CIN_NOTICE_UNAVAILABLE", "IA_UNAUTHENTICATED", "INVALID_INPUT"} {
		require.Contains(t, string(raw), code)
	}
}

// TestCandidateInterviewEndpointDocumentAndChangelog verifies the endpoint document and changelog describe the mounted routes.
func TestCandidateInterviewEndpointDocumentAndChangelog(t *testing.T) {
	endpoint, err := os.ReadFile(filepath.Join("..", "..", "docs", "api", "candidate-interview.md"))
	require.NoError(t, err)
	for _, needle := range []string{
		"GET `/candidate/interview`", "POST `/candidate/consents`", "GET `/candidate/interview/page`",
		"X-Interview-Controller", "X-CSRF-Token", "CIN_FORBIDDEN", "CIN_NOTICE_UNAVAILABLE", "IA_UNAUTHENTICATED", "201", "200",
	} {
		require.Contains(t, string(endpoint), needle)
	}
	changelog, err := os.ReadFile(filepath.Join("..", "..", "docs", "api", "CHANGELOG.md"))
	require.NoError(t, err)
	for _, needle := range []string{"/candidate/interview", "/candidate/consents", "candidate-interview.md"} {
		require.Contains(t, string(changelog), needle)
	}
}

// propertyNames lists the property names of an object schema.
func propertyNames(schema *openapi3.Schema) []string {
	names := make([]string, 0, len(schema.Properties))
	for name := range schema.Properties {
		names = append(names, name)
	}
	return names
}

// schemaAdditional renders whether an object schema forbids additional properties.
func schemaAdditional(schema *openapi3.Schema) string {
	if schema.AdditionalProperties.Has != nil && !*schema.AdditionalProperties.Has {
		return "false"
	}
	return "open"
}
