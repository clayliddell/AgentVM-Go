package crossimport

import (
	"go/ast"
	"strings"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

const doc = "R1: reject cross-imports between feature packages"

var Analyzer = &analysis.Analyzer{
	Name: "crossimport",
	Doc:  doc,
	Run:  run,
	Requires: []*analysis.Analyzer{
		inspect.Analyzer,
	},
}

func run(pass *analysis.Pass) (interface{}, error) {
	inspect := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)

	nodeFilter := []ast.Node{
		(*ast.ImportSpec)(nil),
	}

	inspect.Preorder(nodeFilter, func(n ast.Node) {
		imp := n.(*ast.ImportSpec)
		path := strings.Trim(imp.Path.Value, `"`)

		if strings.Contains(path, "internal/features/") && !strings.HasPrefix(path, pass.Pkg.Path()) {
			pkgPath := pass.Pkg.Path()
			if strings.Contains(pkgPath, "internal/features/") {
				pass.Reportf(imp.Pos(), "R1: feature package %q must not import %q", pkgPath, path)
			}
		}
	})

	return nil, nil
}
