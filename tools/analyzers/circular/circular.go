package circular

import (
	"go/ast"
	"go/types"
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
	directImports := make(map[string]*types.Package)
	for _, imp := range pass.Pkg.Imports() {
		directImports[imp.Path()] = imp
	}

	seen := make(map[string]bool)

	inspect.Preorder(nodeFilter, func(n ast.Node) {
		imp := n.(*ast.ImportSpec)
		path := strings.Trim(imp.Path.Value, `"`)

		if seen[path] {
			return
		}
		seen[path] = true

		importedPkg := directImports[path]
		if importedPkg != nil && importsReachTarget(importedPkg, pkgPath, map[string]bool{pkgPath: true}) {
			pass.Reportf(imp.Pos(), "R8: potential circular dependency detected: %q imports %q — see docs/ARCHITECTURE.md#R8", pkgPath, path)
		}
	})

	return nil, nil
}

func importsReachTarget(pkg *types.Package, target string, seen map[string]bool) bool {
	if pkg == nil {
		return false
	}
	if pkg.Path() == target {
		return true
	}
	if seen[pkg.Path()] {
		return false
	}
	seen[pkg.Path()] = true

	for _, imp := range pkg.Imports() {
		if importsReachTarget(imp, target, seen) {
			return true
		}
	}

	return false
}
