package reexport

import (
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"path/filepath"
	"testing"

	"golang.org/x/tools/go/analysis"
)

func TestStdLibAndModuleHelpers(t *testing.T) {
	if !isStdLib("fmt") {
		t.Fatal("expected fmt to be treated as stdlib")
	}
	if isStdLib("example.com/foo") {
		t.Fatal("expected module path to not be treated as stdlib")
	}
	if !isModulePackage("example.com/foo") {
		t.Fatal("expected module path to be treated as module package")
	}
	if isModulePackage("fmt") {
		t.Fatal("expected fmt to not be treated as module package")
	}
}

func runReexport(t *testing.T, pkgPath, src string, objPkgPath string, objName string) []string {
	t.Helper()
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, filepath.Base(pkgPath)+".go", src, parser.ParseComments)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	var sel *ast.SelectorExpr
	for _, decl := range file.Decls {
		if gen, ok := decl.(*ast.GenDecl); ok {
			for _, spec := range gen.Specs {
				if ts, ok := spec.(*ast.TypeSpec); ok {
					if s, ok := ts.Type.(*ast.SelectorExpr); ok {
						sel = s
						break
					}
				}
			}
		}
	}

	msgs := []string{}

	if sel == nil {
		if objPkgPath != "" || objName != "" {
			t.Fatal("expected selector expression in source")
		}
		pass := &analysis.Pass{
			Fset:  fset,
			Files: []*ast.File{file},
			Pkg:   types.NewPackage(pkgPath, filepath.Base(pkgPath)),
			Report: func(d analysis.Diagnostic) {
				msgs = append(msgs, d.Message)
			},
		}

		if _, err := run(pass); err != nil {
			t.Fatalf("run returned error: %v", err)
		}
		return msgs
	}

	objPkg := types.NewPackage(objPkgPath, filepath.Base(objPkgPath))
	obj := types.NewTypeName(token.NoPos, objPkg, objName, types.Typ[types.Int])
	info := &types.Info{Uses: map[*ast.Ident]types.Object{sel.Sel: obj}}
	pass := &analysis.Pass{
		Fset:      fset,
		Files:     []*ast.File{file},
		Pkg:       types.NewPackage(pkgPath, filepath.Base(pkgPath)),
		TypesInfo: info,
		Report: func(d analysis.Diagnostic) {
			msgs = append(msgs, d.Message)
		},
	}

	if _, err := run(pass); err != nil {
		t.Fatalf("run returned error: %v", err)
	}
	return msgs
}

func TestRunReportsExternalAliasesButNotLocalOrStdlib(t *testing.T) {
	t.Run("external module alias reports", func(t *testing.T) {
		msgs := runReexport(t, "example.com/project/reexport", `package reexport

import ext "example.com/external/pkg"

type Exported = ext.Thing
`, "example.com/external/pkg", "Thing")
		if len(msgs) != 1 {
			t.Fatalf("expected one diagnostic for external alias, got %d: %v", len(msgs), msgs)
		}
	})

	t.Run("local alias does not report", func(t *testing.T) {
		msgs := runReexport(t, "example.com/project/reexport", `package reexport

	type Local struct{}
	type Exported = Local
`, "", "")
		if len(msgs) != 0 {
			t.Fatalf("expected no diagnostic for local alias, got %v", msgs)
		}
	})

	t.Run("stdlib alias does not report", func(t *testing.T) {
		msgs := runReexport(t, "example.com/project/reexport", `package reexport

import "fmt"

type Exported = fmt.Stringer
`, "fmt", "Stringer")
		if len(msgs) != 0 {
			t.Fatalf("expected no diagnostic for stdlib alias, got %v", msgs)
		}
	})
}
