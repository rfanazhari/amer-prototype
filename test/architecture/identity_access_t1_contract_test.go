package architecture_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// identityAccessT1ContractCase names one required administrative-contract artifact and the markers it must contain.
type identityAccessT1ContractCase struct {
	name    string
	relPath string
	needles []string
}

// TestIdentityAccessAdministrativeContract requires T1 CORE-IAM-001 executable
// administrative contracts, Q1 resolution, examples and dependent-task gates.
func TestIdentityAccessAdministrativeContract(t *testing.T) {
	root := filepath.Join("..", "..")
	cases := []identityAccessT1ContractCase{
		{
			name:    "design records resolved Q1 administrative schemas",
			relPath: filepath.Join("docs", "services", "core-ai", "identity-access", "technical-design.md"),
			needles: []string{
				"### 6.1 Administrative identity, password and bootstrap (approved A)",
				"### 6.2 Cookie, origin, CSRF and rate budgets (approved B)",
				"### 6.3 Administrative operation schemas (approved C)",
				"### 6.4 Duplicate accounts and existing bootstrap state (approved D)",
				"### 6.5 Public errors and precedence (approved E)",
				"ACCESS_BOOTSTRAP_ADMIN_EMAIL",
				"ACCESS_BOOTSTRAP_ADMIN_PASSWORD",
				"ACCESS_BOOTSTRAP_ADMIN_NAME",
				"ACCESS_HTTP_ORIGIN",
				"ACCESS_CSRF_SECRET",
				"__Host-ia-admin",
				"X-CSRF-Token",
				"IA_ACCOUNT_CONFLICT",
				"IA_RATE_LIMITED",
				"**Resolved 2026-11-01:**",
			},
		},
		{
			name:    "decision log records T1 contract closure",
			relPath: filepath.Join("docs", "services", "core-ai", "identity-access", "decision-log.md"),
			needles: []string{
				"## D7: Administrative input, provisioning and security contracts are approved",
				"CORE-IAM-001",
				"out-of-band",
			},
		},
		{
			name:    "task plan marks T1 done and unblocks administrative dependents",
			relPath: filepath.Join("docs", "services", "core-ai", "identity-access", "task-plan.md"),
			needles: []string{
				"- [x] **T1. Finalize administrative input, provisioning and security contracts**",
				"Q1 resolved in T1",
			},
		},
		{
			name:    "capability records resolved password and email policy",
			relPath: filepath.Join("docs", "services", "core-ai", "business", "capabilities", "identity-access.md"),
			needles: []string{
				"Administrative email is canonicalized by trimming outer whitespace and Unicode lowercasing the entire address",
				"Initial HR passwords are chosen by ADMIN at provisioning and delivered out of band",
			},
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
	t.Run("login and provision examples are valid JSON objects", func(t *testing.T) {
		body, err := os.ReadFile(filepath.Join(root, "docs", "services", "core-ai", "identity-access", "technical-design.md"))
		require.NoError(t, err)
		examples := map[string][]string{
			"login request":        {"```json", `"email"`, `"password"`},
			"hr provision request": {"```json", `"name"`, `"email"`, `"password"`},
		}
		text := string(body)
		require.True(t, strings.Contains(text, "#### Examples (accepted expectations)"), "missing accepted JSON examples heading")
		for name, required := range examples {
			assert.Truef(t, jsonExampleContains(text, required...), "%s example missing required members", name)
		}
		for _, block := range jsonFencedBlocks(text) {
			var value any
			require.NoError(t, json.Unmarshal([]byte(block), &value), block)
			_, isObject := value.(map[string]any)
			require.True(t, isObject, block)
		}
	})
}

// jsonExampleContains reports whether any fenced JSON object in text includes all needles.
func jsonExampleContains(text string, needles ...string) bool {
	for _, block := range jsonFencedBlocks(text) {
		ok := true
		for _, needle := range needles {
			if !strings.Contains(block, needle) {
				ok = false
				break
			}
		}
		if ok {
			return true
		}
	}
	return false
}

// jsonFencedBlocks returns fenced json code samples from markdown.
func jsonFencedBlocks(text string) []string {
	const fence = "```json"
	var blocks []string
	remaining := text
	for {
		start := strings.Index(remaining, fence)
		if start < 0 {
			return blocks
		}
		remaining = remaining[start+len(fence):]
		end := strings.Index(remaining, "```")
		if end < 0 {
			return blocks
		}
		blocks = append(blocks, strings.TrimSpace(remaining[:end]))
		remaining = remaining[end+3:]
	}
}
