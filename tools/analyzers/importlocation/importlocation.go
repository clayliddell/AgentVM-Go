package importlocation

import (
	"go/ast"
	"path/filepath"
	"strings"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

const doc = "R11/R12: restrict database/sql to store.go and net/http to handler.go"

var Analyzer = &analysis.Analyzer{
	Name: "importlocation",
	Doc:  doc,
	Run:  run,
	Requires: []*analysis.Analyzer{
		inspect.Analyzer,
	},
}

var restrictedImports = map[string]string{
	"database/sql": "store.go",
	"net/http":     "handler.go",
}

func run(pass *analysis.Pass) (interface{}, error) {
	fset := pass.Fset

	for _, file := range pass.Files {
		filename := fset.Position(file.Pos()).Filename
		baseName := filepath.Base(filename)

		inspect := inspector.New([]*ast.File{file})

		nodeFilter := []ast.Node{
			(*ast.ImportSpec)(nil),
		}

		inspect.Preorder(nodeFilter, func(n ast.Node) {
			imp := n.(*ast.ImportSpec)
			path := strings.Trim(imp.Path.Value, `"`)

			if allowedFile, ok := restrictedImports[path]; ok {
				if baseName != allowedFile {
					pass.Reportf(imp.Pos(), "R11/R12: %q may only be imported in %s files, found in %s", path, allowedFile, baseName)
				}
			}
		})
	}

	return nil, nil
}
