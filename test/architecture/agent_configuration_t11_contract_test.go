package architecture_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// agentConfigurationT11ContractCase names one required assignment/execution binding artifact marker.
type agentConfigurationT11ContractCase struct {
	name    string
	relPath string
	needles []string
}

// TestAgentConfigurationBindingConsumers requires T11 CORE-ACP-011 pinned assignment and execution consumers.
func TestAgentConfigurationBindingConsumers(t *testing.T) {
	root := filepath.Join("..", "..")
	cases := []agentConfigurationT11ContractCase{
		{
			name:    "application pin assignment use case exists",
			relPath: filepath.Join("application", "usecase", "agentbindingusecase", "pin_assignment.go"),
			needles: []string{"type PinAssignment struct", "FindInstance"},
		},
		{
			name:    "application start execution use case exists",
			relPath: filepath.Join("application", "usecase", "agentbindingusecase", "start_execution.go"),
			needles: []string{"type StartExecution struct", "FindRevision"},
		},
		{
			name:    "sql adapter writes interview assignment and execution pins",
			relPath: filepath.Join("infrastructure", "repository", "agentbindingrepository", "store.go"),
			needles: []string{"interview_assignments", "interview_executions", "published_revision_id"},
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
