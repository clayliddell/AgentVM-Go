package mutablestate

import (
	"go/ast"
	"go/token"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

const doc = "R9: flag package-level mutable state (top-level var declarations)"

var Analyzer = &analysis.Analyzer{
	Name: "mutablestate",
	Doc:  doc,
	Run:  run,
	Requires: []*analysis.Analyzer{
		inspect.Analyzer,
	},
}

func run(pass *analysis.Pass) (interface{}, error) {
	inspect := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)

	nodeFilter := []ast.Node{
		(*ast.GenDecl)(nil),
	}

	inspect.Preorder(nodeFilter, func(n ast.Node) {
		decl := n.(*ast.GenDecl)
		if decl.Tok != token.VAR {
			return
		}

		for _, spec := range decl.Specs {
			vs, ok := spec.(*ast.ValueSpec)
			if !ok {
				continue
			}

			for _, name := range vs.Names {
				if name.IsExported() && name.Name != "Analyzer" {
					pass.Reportf(name.Pos(), "R9: package-level mutable state %q is banned", name.Name)
				}
			}
		}
	})

	return nil, nil
}
