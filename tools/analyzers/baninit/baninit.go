package baninit

import (
	"go/ast"
	"path/filepath"
	"strings"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

const doc = "R10: ban init() functions"

var Analyzer = &analysis.Analyzer{
	Name: "baninit",
	Doc:  doc,
	Run:  run,
	Requires: []*analysis.Analyzer{
		inspect.Analyzer,
	},
}

func run(pass *analysis.Pass) (interface{}, error) {
	inspect := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)

	nodeFilter := []ast.Node{
		(*ast.FuncDecl)(nil),
	}

	inspect.Preorder(nodeFilter, func(n ast.Node) {
		fn := n.(*ast.FuncDecl)
		if fn.Name.Name == "init" && fn.Recv == nil {
			fset := pass.Fset
			filename := fset.Position(fn.Pos()).Filename
			baseName := filepath.Base(filename)
			dir := filepath.Dir(filename)

			if strings.HasSuffix(baseName, "_test.go") {
				return
			}

			if !strings.Contains(dir, "internal/") {
				return
			}

			pass.Reportf(fn.Pos(), "R10: init() functions are banned")
		}
	})

	return nil, nil
}
