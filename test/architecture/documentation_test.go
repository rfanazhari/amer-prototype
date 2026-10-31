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

// TestDeclarationDocumentation checks the owner's documentation rule for all authored Go declarations and interface methods.
func TestDeclarationDocumentation(t *testing.T) {
	root := filepath.Join("..", "..")
	for _, directory := range []string{"domain", "application", "delivery", "infrastructure", "cmd", "pkg", "translations", "test"} {
		err := filepath.WalkDir(filepath.Join(root, directory), func(path string, entry fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if entry.IsDir() || !strings.HasSuffix(path, ".go") {
				return nil
			}
			file, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ParseComments)
			if err != nil {
				return err
			}
			if ast.IsGenerated(file) {
				return nil
			}
			ast.Inspect(file, func(node ast.Node) bool {
				switch declaration := node.(type) {
				case *ast.FuncDecl:
					assert.NotNil(t, declaration.Doc, "missing function/method documentation: %s in %s", declaration.Name.Name, path)
				case *ast.GenDecl:
					for _, spec := range declaration.Specs {
						if definition, ok := spec.(*ast.TypeSpec); ok {
							assert.True(t, definition.Doc != nil || declaration.Doc != nil, "missing type documentation: %s in %s", definition.Name.Name, path)
						}
					}
				case *ast.InterfaceType:
					for _, method := range declaration.Methods.List {
						if len(method.Names) > 0 {
							assert.NotNil(t, method.Doc, "missing interface method documentation: %s in %s", method.Names[0].Name, path)
						}
					}
				}
				return true
			})
			return nil
		})
		require.NoError(t, err)
	}
}
