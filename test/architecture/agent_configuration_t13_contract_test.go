package architecture_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// agentConfigurationT13ContractCase names one required assignment-create artifact marker.
type agentConfigurationT13ContractCase struct {
	name    string
	relPath string
	needles []string
}

// TestAgentConfigurationAssignmentCreateContract requires T13 CORE-ACP-013 public assignment create.
func TestAgentConfigurationAssignmentCreateContract(t *testing.T) {
	root := filepath.Join("..", "..")
	cases := []agentConfigurationT13ContractCase{
		{
			name:    "application create assignment use case exists",
			relPath: filepath.Join("application", "usecase", "agentbindingusecase", "create_assignment.go"),
			needles: []string{"type CreateAssignment struct", "FindByID", "hr-interviewer"},
		},
		{
			name:    "access handler mounts assignment collection create",
			relPath: filepath.Join("delivery", "http", "handler", "accesshandler", "candidate.go"),
			needles: []string{"createAssignment", `"/admin/interview-assignments"`},
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
