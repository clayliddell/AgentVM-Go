package sharedtypes

import (
	"go/ast"
	"strings"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

const doc = "R2: reject feature imports in shared/types packages"

var Analyzer = &analysis.Analyzer{
	Name: "sharedtypes",
	Doc:  doc,
	Run:  run,
	Requires: []*analysis.Analyzer{
		inspect.Analyzer,
	},
}

func run(pass *analysis.Pass) (interface{}, error) {
	pkgPath := pass.Pkg.Path()
	if !strings.Contains(pkgPath, "shared/types") {
		return nil, nil
	}

	inspect := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)

	nodeFilter := []ast.Node{
		(*ast.ImportSpec)(nil),
	}

	inspect.Preorder(nodeFilter, func(n ast.Node) {
		imp := n.(*ast.ImportSpec)
		path := strings.Trim(imp.Path.Value, `"`)

		if strings.Contains(path, "features/") {
			pass.Reportf(imp.Pos(), "R2: shared/types must not import feature packages — see docs/ARCHITECTURE.md#R2")
		}
	})

	return nil, nil
}
