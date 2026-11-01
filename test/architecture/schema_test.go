package architecture_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSchemaHasOneOwner prevents executable AutoMigrate calls in runtime and test setup.
func TestSchemaHasOneOwner(t *testing.T) {
	for _, directory := range []string{"infrastructure", "cmd", "test"} {
		err := filepath.WalkDir(filepath.Join("../..", directory), func(path string, entry fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if entry.IsDir() || !strings.HasSuffix(path, ".go") {
				return nil
			}
			file, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
			if err != nil {
				return err
			}
			ast.Inspect(file, func(node ast.Node) bool {
				if selector, ok := node.(*ast.SelectorExpr); ok {
					assert.NotEqual(t, "AutoMigrate", selector.Sel.Name, "schema DDL has another owner in %s", path)
				}
				return true
			})
			return nil
		})
		require.NoError(t, err)
	}
}
