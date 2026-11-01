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

// identityAccessT10ContractCase names one required candidate-contract artifact and the markers it must contain.
type identityAccessT10ContractCase struct {
	name    string
	relPath string
	needles []string
}

// TestIdentityAccessCandidateContract requires T10 CORE-IAM-010 executable
// candidate, exchange and assignment collaboration contracts with Q2/Q3/Q4 resolved.
func TestIdentityAccessCandidateContract(t *testing.T) {
	root := filepath.Join("..", "..")
	cases := []identityAccessT10ContractCase{
		{
			name:    "design records resolved Q2 Q3 Q4 candidate schemas",
			relPath: filepath.Join("docs", "services", "core-ai", "identity-access", "technical-design.md"),
			needles: []string{
				"### 6.6 Candidate contact and matching (approved F)",
				"### 6.7 Invitation exchange reuse and recovery (approved G)",
				"### 6.8 Candidate operation schemas (approved H)",
				"### 6.9 Assignment collaboration and locking (approved I)",
				"interview_assignments",
				"no consumed_at",
				"__Host-ia-candidate",
				"exactly `token`",
				"**Resolved 2026-11-01 (T10):** Q2",
			},
		},
		{
			name:    "decision log records T10 contract closure",
			relPath: filepath.Join("docs", "services", "core-ai", "identity-access", "decision-log.md"),
			needles: []string{
				"## D9: Candidate contact, exchange reuse and assignment collaboration contracts are approved",
				"CORE-IAM-010",
				"bearer link proves possession",
			},
		},
		{
			name:    "task plan marks T10 done and unblocks candidate dependents",
			relPath: filepath.Join("docs", "services", "core-ai", "identity-access", "task-plan.md"),
			needles: []string{
				"- [x] **T10. Resolve candidate, exchange and actual assignment collaboration contracts**",
				"Q2/Q3/Q4 resolved in T10",
			},
		},
		{
			name:    "capability records resolved candidate contact and exchange policy",
			relPath: filepath.Join("docs", "services", "core-ai", "business", "capabilities", "identity-access.md"),
			needles: []string{
				"Candidate email uses the same canonicalization as administrative email and is never unique or used for upsert",
				"A valid invitation may be exchanged repeatedly until expiry or revocation",
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
	t.Run("candidate and exchange examples are valid JSON objects", func(t *testing.T) {
		body, err := os.ReadFile(filepath.Join(root, "docs", "services", "core-ai", "identity-access", "technical-design.md"))
		require.NoError(t, err)
		text := string(body)
		require.True(t, strings.Contains(text, "#### Candidate examples (accepted expectations)"), "missing candidate JSON examples heading")
		assert.True(t, jsonExampleContains(text, `"name":" Andi Pratama "`, `"email":"Andi@Example.com"`), "candidate create request example missing required members")
		assert.True(t, jsonExampleContains(text, `"token":"invitation-secret"`), "exchange request example missing token")
		for _, block := range jsonFencedBlocks(text) {
			var value any
			require.NoError(t, json.Unmarshal([]byte(block), &value), block)
			_, isObject := value.(map[string]any)
			require.True(t, isObject, block)
		}
	})
}
