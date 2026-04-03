package circular

import (
	"go/ast"
	"strings"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

const doc = "R8: detect circular dependencies between packages"

var Analyzer = &analysis.Analyzer{
	Name: "circular",
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

	pkgPath := pass.Pkg.Path()
	if strings.HasSuffix(pkgPath, ".test") {
		return nil, nil
	}

	seen := make(map[string]bool)

	inspect.Preorder(nodeFilter, func(n ast.Node) {
		imp := n.(*ast.ImportSpec)
		path := strings.Trim(imp.Path.Value, `"`)

		if seen[path] {
			return
		}
		seen[path] = true

		if strings.HasPrefix(path, pkgPath+".") || strings.HasPrefix(pkgPath, path+".") {
			pass.Reportf(imp.Pos(), "R8: potential circular dependency detected: %q imports %q — see docs/ARCHITECTURE.md#R8", pkgPath, path)
		}
	})

	return nil, nil
}
