package architecture_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"strings"
)

// scanKernelIdentifiers reports every identifier in the non-test Go files of dirs, relative to root, that contains a
// forbidden term case-insensitively. Each result names the file, the identifier and the matched term.
func scanKernelIdentifiers(root string, dirs, forbidden []string) ([]string, error) {
	var found []string
	for _, dir := range dirs {
		fileSet := token.NewFileSet()
		err := filepath.WalkDir(filepath.Join(root, dir), func(path string, entry fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if entry.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return nil
			}
			file, err := parser.ParseFile(fileSet, path, nil, 0)
			if err != nil {
				return err
			}
			ast.Inspect(file, func(node ast.Node) bool {
				ident, ok := node.(*ast.Ident)
				if !ok {
					return true
				}
				lower := strings.ToLower(ident.Name)
				for _, term := range forbidden {
					if strings.Contains(lower, term) {
						found = append(found, path+": "+ident.Name+" contains "+term)
					}
				}
				return true
			})
			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	return found, nil
}
