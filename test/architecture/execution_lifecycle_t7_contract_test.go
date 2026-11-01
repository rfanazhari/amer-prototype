package architecture_test

import (
	"go/parser"
	"go/token"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestInvitationIssueDoesNotCallExecutionCancellation proves BR-CORE-037: reissuing an invitation never reaches
// the execution lifecycle, so a replacement link cannot cancel a run that is already underway.
func TestInvitationIssueDoesNotCallExecutionCancellation(t *testing.T) {
	file, err := parser.ParseFile(token.NewFileSet(), "../../application/usecase/accessusecase/issue_invitation.go", nil, parser.ImportsOnly)
	require.NoError(t, err)

	for _, spec := range file.Imports {
		path, err := strconv.Unquote(spec.Path.Value)
		require.NoError(t, err)
		require.False(t, strings.Contains(path, "executionusecase") || strings.Contains(path, "executionport") ||
			strings.Contains(path, "domain/aggregate/execution"), "issue_invitation.go must not import %s", path)
	}
}
