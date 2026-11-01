package architecture_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// identityAccessT13ContractCase names one published candidate HTTP contract artifact.
type identityAccessT13ContractCase struct {
	name    string
	relPath string
	needles []string
}

// TestIdentityAccessCandidateHTTPContract requires T13 published candidate routes, cookies and changelog.
func TestIdentityAccessCandidateHTTPContract(t *testing.T) {
	root := filepath.Join("..", "..")
	cases := []identityAccessT13ContractCase{
		{
			name:    "openapi publishes candidate identity invitation and session paths",
			relPath: filepath.Join("docs", "api", "openapi.yaml"),
			needles: []string{
				"/admin/candidates",
				"/admin/candidates/{id}",
				"/admin/interview-assignments/{id}/invitations",
				"/candidate/access/exchange",
				"/candidate/session",
				"/candidate/access",
				"IA_INVITATION_UNAVAILABLE",
				"CandidateCookie",
			},
		},
		{
			name:    "identity-access endpoint document covers candidate HTTP",
			relPath: filepath.Join("docs", "api", "identity-access.md"),
			needles: []string{
				"POST `/admin/candidates`",
				"GET `/admin/candidates/{id}`",
				"POST `/admin/interview-assignments/{id}/invitations`",
				"POST `/candidate/access/exchange`",
				"GET `/candidate/session`",
				"GET `/candidate/access`",
				"history.replaceState",
				"no-referrer",
				"GET/prefetch cannot exchange",
			},
		},
		{
			name:    "changelog records candidate access",
			relPath: filepath.Join("docs", "api", "CHANGELOG.md"),
			needles: []string{"identity-access.md", "/candidate/access/exchange", "/admin/candidates"},
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
