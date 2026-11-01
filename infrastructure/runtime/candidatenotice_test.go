package runtime_test

import (
	"os"
	"path/filepath"
	"regexp"
	"testing"

	appruntime "github.com/rfanazhari/amer-prototype/infrastructure/runtime"
	"github.com/stretchr/testify/require"
)

// noticeVersionPattern is the character rule for a notice version in technical design section 3.
var noticeVersionPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:-]{0,63}$`)

// TestCurrentCandidateNoticeIsApproved verifies the composed catalog is not empty and uses the approved version.
func TestCurrentCandidateNoticeIsApproved(t *testing.T) {
	notice := appruntime.CurrentCandidateNotice()

	require.Equal(t, "notice-261102", notice.Version)
	require.Regexp(t, noticeVersionPattern, notice.Version)
	require.NotEmpty(t, notice.Statement)
}

// TestCurrentCandidateNoticeMatchesDesignRecord verifies design section 6 names the exact version and full statement.
func TestCurrentCandidateNoticeMatchesDesignRecord(t *testing.T) {
	notice := appruntime.CurrentCandidateNotice()
	design, err := os.ReadFile(filepath.Join("..", "..", "docs", "services", "core-ai", "candidate-interview", "technical-design.md"))
	require.NoError(t, err)

	require.NotEmpty(t, notice.Version)
	require.NotEmpty(t, notice.Statement)
	require.Contains(t, string(design), notice.Version)
	require.Contains(t, string(design), notice.Statement)
}
