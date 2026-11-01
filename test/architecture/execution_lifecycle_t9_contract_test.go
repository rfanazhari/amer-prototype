package architecture_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// kernelDirs lists the execution lifecycle kernel packages named in technical design section 4.
var kernelDirs = []string{
	"domain/aggregate/execution",
	"application/usecase/executionusecase",
	"application/query/executionquery",
	"application/repository/executionport",
	"infrastructure/repository/executionrepository",
}

// kernelForbiddenTerms lists the product vocabulary the shared lifecycle must not name (decision D6).
var kernelForbiddenTerms = []string{"stage", "invoice", "transcript", "salary"}

// TestKernelScanReportsForbiddenIdentifiers proves the scan itself fails on a violating identifier and ignores test files.
func TestKernelScanReportsForbiddenIdentifiers(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "kernel")
	require.NoError(t, os.MkdirAll(dir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "bad.go"), []byte("package kernel\n\n// InvoiceTotal is a violation.\nfunc InvoiceTotal() {}\n"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "good.go"), []byte("package kernel\n\n// Run is clean.\nfunc Run() {}\n"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "ignored_test.go"), []byte("package kernel\n\n// SalaryFixture is only a test name.\nfunc SalaryFixture() {}\n"), 0o600))

	found, err := scanKernelIdentifiers(root, []string{"kernel"}, kernelForbiddenTerms)

	require.NoError(t, err)
	require.Len(t, found, 1)
	require.Contains(t, found[0], "bad.go")
	require.Contains(t, found[0], "InvoiceTotal")
}

// TestExecutionKernelHasNoProductIdentifiers enforces decision D6 on the real kernel packages.
func TestExecutionKernelHasNoProductIdentifiers(t *testing.T) {
	found, err := scanKernelIdentifiers(filepath.Join("..", ".."), kernelDirs, kernelForbiddenTerms)

	require.NoError(t, err)
	require.Empty(t, found)
}

// TestRuntimeDoesNotMountExecutionLifecycle proves the runtime composition neither calls StartExecution nor adds a lifecycle route.
func TestRuntimeDoesNotMountExecutionLifecycle(t *testing.T) {
	files, err := filepath.Glob(filepath.Join("..", "..", "infrastructure", "runtime", "*.go"))
	require.NoError(t, err)
	require.NotEmpty(t, files)
	for _, file := range files {
		if strings.HasSuffix(file, "_test.go") {
			continue
		}
		body, err := os.ReadFile(file)
		require.NoError(t, err)
		require.NotContains(t, string(body), "StartExecution", file)
		require.NotContains(t, string(body), "executionusecase", file)
		require.NotContains(t, string(body), "executionquery", file)
	}
}

// TestOpenAPIHasNoExecutionRoute proves the public contract gained no execution route.
func TestOpenAPIHasNoExecutionRoute(t *testing.T) {
	body, err := os.ReadFile(filepath.Join("..", "..", "docs", "api", "openapi.yaml"))
	require.NoError(t, err)
	for _, line := range strings.Split(string(body), "\n") {
		if strings.HasPrefix(line, "  /") {
			require.NotContains(t, strings.ToLower(line), "execution", line)
		}
	}
}
