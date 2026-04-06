package reexport

import (
	"go/ast"
	"go/token"
	"strings"

	"golang.org/x/tools/go/analysis"
)

const doc = "R19: no re-exporting — packages must not export types from other packages"

var Analyzer = &analysis.Analyzer{
	Name: "reexport",
	Doc:  doc,
	Run:  run,
}

func run(pass *analysis.Pass) (interface{}, error) {
	for _, file := range pass.Files {
		for _, decl := range file.Decls {
			genDecl, ok := decl.(*ast.GenDecl)
			if !ok || genDecl.Tok != token.TYPE {
				continue
			}

			for _, spec := range genDecl.Specs {
				ts, ok := spec.(*ast.TypeSpec)
				if !ok || !ts.Name.IsExported() {
					continue
				}

				if !ts.Assign.IsValid() {
					continue
				}

				sel, ok := ts.Type.(*ast.SelectorExpr)
				if !ok {
					continue
				}

				obj, ok := pass.TypesInfo.Uses[sel.Sel]
				if !ok || obj == nil {
					continue
				}

				objPkg := obj.Pkg()
				if objPkg == nil {
					continue
				}

				objPkgPath := objPkg.Path()
				if objPkgPath != pass.Pkg.Path() && isModulePackage(objPkgPath) {
					pass.Reportf(ts.Pos(), "R19: %q aliases type %q from %q — see docs/ARCHITECTURE.md#R19", pass.Pkg.Path(), ts.Name.Name, objPkgPath)
				}
			}
		}
	}

	return nil, nil
}

func isModulePackage(pkgPath string) bool {
	return !isStdLib(pkgPath)
}

func isStdLib(pkgPath string) bool {
	if strings.Contains(pkgPath, ".") {
		return false
	}
	parts := strings.Split(pkgPath, "/")
	if len(parts) == 0 {
		return true
	}
	first := parts[0]
	stdLib := map[string]bool{
		"fmt": true, "strings": true, "bytes": true, "io": true, "os": true,
		"net": true, "time": true, "context": true, "errors": true, "sync": true,
		"sort": true, "math": true, "strconv": true, "encoding": true, "reflect": true,
		"testing": true, "runtime": true, "path": true, "log": true, "flag": true,
		"syscall": true, "unsafe": true, "unicode": true, "regexp": true,
	}
	return stdLib[first]
}
