package circular

import (
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"testing"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

func TestImportsReachTarget(t *testing.T) {
	target := "example.com/a"
	pkga := types.NewPackage(target, "a")
	pkgb := types.NewPackage("example.com/b", "b")
	pkgc := types.NewPackage("example.com/c", "c")
	pkga.SetImports([]*types.Package{pkgb})
	pkgb.SetImports([]*types.Package{pkgc})
	pkgc.SetImports([]*types.Package{pkga})

	if !importsReachTarget(pkga, target, map[string]bool{target: true}) {
		t.Fatal("expected direct target match to return true")
	}
	if !importsReachTarget(pkgb, target, map[string]bool{target: true}) {
		t.Fatal("expected indirect cycle to return true")
	}
	if importsReachTarget(nil, target, map[string]bool{}) {
		t.Fatal("expected nil package to return false")
	}

	pkgx := types.NewPackage("example.com/x", "x")
	pkgy := types.NewPackage("example.com/y", "y")
	pkgx.SetImports([]*types.Package{pkgy})
	pkgy.SetImports([]*types.Package{pkgx})
	if importsReachTarget(pkgx, target, map[string]bool{target: true}) {
		t.Fatal("expected cycle without target to return false")
	}
}

func TestRunReportsCircularDependency(t *testing.T) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "a.go", `package a

import _ "example.com/b"
`, parser.ParseComments)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	pkgA := types.NewPackage("example.com/a", "a")
	pkgB := types.NewPackage("example.com/b", "b")
	pkgA.SetImports([]*types.Package{pkgB})
	pkgB.SetImports([]*types.Package{pkgA})

	msgs := []string{}
	pass := &analysis.Pass{
		Fset:  fset,
		Files: []*ast.File{file},
		Pkg:   pkgA,
		Report: func(d analysis.Diagnostic) {
			msgs = append(msgs, d.Message)
		},
		ResultOf: map[*analysis.Analyzer]any{
			inspect.Analyzer: inspector.New([]*ast.File{file}),
		},
	}

	if _, err := run(pass); err != nil {
		t.Fatalf("run returned error: %v", err)
	}
	if len(msgs) != 1 {
		t.Fatalf("expected one circular dependency diagnostic, got %d: %v", len(msgs), msgs)
	}
}
