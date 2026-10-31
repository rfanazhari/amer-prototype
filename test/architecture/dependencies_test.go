package architecture_test

import (
	"github.com/stretchr/testify/require"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// TestDependencyDirections checks inward import dependencies across the four layers using Go source imports.
func TestDependencyDirections(t *testing.T) {
	root := filepath.Join("..", "..")
	const module = "github.com/rfanazhari/amer-prototype/"
	domainStd := map[string]bool{"errors": true, "time": true, "strings": true, "unicode/utf8": true, "fmt": true}
	for _, layer := range []string{"domain", "application", "delivery", "infrastructure"} {
		err := filepath.WalkDir(filepath.Join(root, layer), func(path string, entry fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if entry.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return nil
			}
			file, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ImportsOnly)
			if err != nil {
				return err
			}
			for _, spec := range file.Imports {
				imported, err := strconv.Unquote(spec.Path.Value)
				require.NoError(t, err)
				if layer == "domain" {
					require.True(t, domainStd[imported] || strings.HasPrefix(imported, module+"domain/"), "domain imports %s in %s", imported, path)
				}
				if layer == "application" {
					require.False(t, strings.HasPrefix(imported, module+"delivery/") || strings.HasPrefix(imported, module+"infrastructure/") || strings.Contains(imported, "gorm.io/"), "application imports %s in %s", imported, path)
				}
				if layer == "delivery" {
					require.False(t, strings.HasPrefix(imported, module+"infrastructure/"), "delivery imports %s in %s", imported, path)
				}
				if layer == "infrastructure" && !strings.Contains(filepath.ToSlash(path), "infrastructure/runtime/") {
					require.False(t, strings.HasPrefix(imported, module+"delivery/"), "adapter imports delivery in %s", path)
				}
			}
			return nil
		})
		require.NoError(t, err)
	}
}
