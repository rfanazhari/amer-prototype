package architecture_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestIdentityAccessACPQtwoEvidence requires CORE-IAM-009 access evidence for ACP Q2.
func TestIdentityAccessACPQtwoEvidence(t *testing.T) {
	root := filepath.Join("..", "..")
	path := filepath.Join(root, "docs", "services", "core-ai", "agent-configuration-publication", "q2-access-evidence.md")
	body, err := os.ReadFile(path)
	require.NoError(t, err)
	text := string(body)
	for _, needle := range []string{
		"CORE-IAM-009",
		"bootstrap",
		"ADMIN configuration",
		"candidate cookie",
		"ACP T8/T9 remain",
	} {
		assert.Truef(t, strings.Contains(text, needle), "missing %q", needle)
	}
}
