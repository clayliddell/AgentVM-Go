package wiringonly

import (
	"go/ast"
	"strings"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

const doc = "R3: only wiring may import multiple feature packages"

var Analyzer = &analysis.Analyzer{
	Name: "wiringonly",
	Doc:  doc,
	Run:  run,
	Requires: []*analysis.Analyzer{
		inspect.Analyzer,
	},
}

func run(pass *analysis.Pass) (interface{}, error) {
	pkgPath := pass.Pkg.Path()
	if strings.Contains(pkgPath, "/wiring") {
		return nil, nil
	}

	inspect := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)

	featureImports := make(map[string]bool)

	nodeFilter := []ast.Node{
		(*ast.ImportSpec)(nil),
	}

	inspect.Preorder(nodeFilter, func(n ast.Node) {
		imp := n.(*ast.ImportSpec)
		path := strings.Trim(imp.Path.Value, `"`)

		if strings.Contains(path, "internal/features/") {
			featureImports[path] = true
		}
	})

	if len(featureImports) > 1 {
		pass.Reportf(0, "R3: only wiring may import multiple feature packages, %q imports %d — see docs/ARCHITECTURE.md#R3", pkgPath, len(featureImports))
	}

	return nil, nil
}
