package architecture_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestIdentityAccessCandidateIntegrationEvidence requires CORE-IAM-014 proof, skips and browser commands.
func TestIdentityAccessCandidateIntegrationEvidence(t *testing.T) {
	root := filepath.Join("..", "..")
	path := filepath.Join(root, "docs", "services", "core-ai", "identity-access", "t14-evidence.md")
	body, err := os.ReadFile(path)
	require.NoError(t, err)
	text := string(body)
	for _, needle := range []string{
		"CORE-IAM-014",
		"assignments A/B",
		"isolated sessions",
		"exchange retry",
		"exact expiry",
		"source-invitation replacement",
		"assignment revocation",
		"completed no-restart",
		"TEST_DATABASE_URL",
		"go test ./delivery/http/candidatelanding",
		"active-run cancellation",
		"Interview consumer not present",
		"make test.integration",
	} {
		assert.Truef(t, strings.Contains(text, needle), "missing %q", needle)
	}
}
