package mutablestate

import (
	"go/ast"
	"go/token"
	"path/filepath"
	"strings"

	"golang.org/x/tools/go/analysis"
)

const doc = "R9: flag package-level mutable state (top-level var declarations)"

var Analyzer = &analysis.Analyzer{
	Name: "mutablestate",
	Doc:  doc,
	Run:  run,
}

func run(pass *analysis.Pass) (interface{}, error) {
	if !strings.Contains(pass.Pkg.Path(), "internal/") {
		return nil, nil
	}

	for _, file := range pass.Files {
		filename := pass.Fset.Position(file.Pos()).Filename
		if !strings.Contains(filename, "/internal/") {
			continue
		}
		if strings.HasSuffix(filepath.Base(filename), "_test.go") {
			continue
		}

		for _, decl := range file.Decls {
			genDecl, ok := decl.(*ast.GenDecl)
			if !ok || genDecl.Tok != token.VAR {
				continue
			}

			for _, spec := range genDecl.Specs {
				vs, ok := spec.(*ast.ValueSpec)
				if !ok {
					continue
				}
				if isEmbeddedFS(vs) {
					continue
				}

				for _, name := range vs.Names {
					if name.Name == "Analyzer" || strings.HasPrefix(name.Name, "err") {
						continue
					}
					pass.Reportf(name.Pos(), "R9: package-level mutable state %q is banned — see docs/ARCHITECTURE.md#R9", name.Name)
				}
			}
		}
	}

	return nil, nil
}

func isEmbeddedFS(vs *ast.ValueSpec) bool {
	sel, ok := vs.Type.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	pkg, ok := sel.X.(*ast.Ident)
	return ok && pkg.Name == "embed" && sel.Sel.Name == "FS"
}
