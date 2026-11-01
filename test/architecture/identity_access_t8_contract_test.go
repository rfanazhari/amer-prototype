package architecture_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// identityAccessHTTPContractCase names one published administrative HTTP contract artifact.
type identityAccessHTTPContractCase struct {
	name    string
	relPath string
	needles []string
}

// TestIdentityAccessAdministrativeHTTPContract requires T8 published routes, errors and changelog.
func TestIdentityAccessAdministrativeHTTPContract(t *testing.T) {
	root := filepath.Join("..", "..")
	cases := []identityAccessHTTPContractCase{
		{
			name:    "openapi publishes four administrative paths",
			relPath: filepath.Join("docs", "api", "openapi.yaml"),
			needles: []string{"/admin/sessions", "/admin/session", "/admin/sessions/current", "/admin/hr-accounts", "IA_UNAUTHENTICATED", "IA_RATE_LIMITED", "IA_ACCOUNT_CONFLICT"},
		},
		{
			name:    "identity-access endpoint document exists",
			relPath: filepath.Join("docs", "api", "identity-access.md"),
			needles: []string{"POST `/admin/sessions`", "GET `/admin/session`", "DELETE `/admin/sessions/current`", "POST `/admin/hr-accounts`", "IA_FORBIDDEN", "no-store"},
		},
		{
			name:    "changelog records administrative access",
			relPath: filepath.Join("docs", "api", "CHANGELOG.md"),
			needles: []string{"identity-access.md", "/admin/sessions"},
		},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			body, err := os.ReadFile(filepath.Join(root, tt.relPath))
			require.NoError(t, err)
			text := string(body)
			for _, needle := range tt.needles {
				assert.Truef(t, strings.Contains(text, needle), "missing %q in %s", needle, tt.relPath)
			}
		})
	}
}
