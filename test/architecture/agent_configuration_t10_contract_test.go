package architecture_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// agentConfigurationT10ContractCase names one required deployment-compatibility artifact marker.
type agentConfigurationT10ContractCase struct {
	name    string
	relPath string
	needles []string
}

// TestAgentConfigurationDeploymentCompatibility requires T10 CORE-ACP-010 restoration policy and Q4 resolution.
func TestAgentConfigurationDeploymentCompatibility(t *testing.T) {
	root := filepath.Join("..", "..")
	cases := []agentConfigurationT10ContractCase{
		{
			name:    "design records retained restoration and restricted deployment policy",
			relPath: filepath.Join("docs", "services", "core-ai", "agent-configuration-publication", "technical-design.md"),
			needles: []string{
				"### 6.6 Historical deployment compatibility (approved T10)",
				"Retain-and-restrict deployment",
				"**Resolved 2026-11-01 (T10):** Q4",
				"RequireAgentConfigurationDeploymentCompatible",
			},
		},
		{
			name:    "decision log records T10 Q4 closure",
			relPath: filepath.Join("docs", "services", "core-ai", "agent-configuration-publication", "decision-log.md"),
			needles: []string{
				"## D8: Historical template deployment compatibility is approved",
				"CORE-ACP-010",
				"without current-creation-policy reinterpretation",
			},
		},
		{
			name:    "task plan marks T10 done",
			relPath: filepath.Join("docs", "services", "core-ai", "agent-configuration-publication", "task-plan.md"),
			needles: []string{
				"- [x] **T10. Establish historical template deployment compatibility**",
				"CORE-ACP-010",
			},
		},
		{
			name:    "approval record maps T10 policy sections",
			relPath: filepath.Join("docs", "services", "core-ai", "agent-configuration-publication", "t10-contract.md"),
			needles: []string{
				"CORE-ACP-010",
				"Retain-and-restrict deployment",
				"Representative fixtures",
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
}
