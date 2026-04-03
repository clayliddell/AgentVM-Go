package ioseparation

import (
	"path/filepath"
	"strings"

	"golang.org/x/tools/go/analysis"
)

const doc = "R18: business logic files must not import IO packages"

var Analyzer = &analysis.Analyzer{
	Name: "ioseparation",
	Doc:  doc,
	Run:  run,
}

var ioPackages = map[string]bool{
	"database/sql": true,
	"os":           true,
	"io":           true,
	"io/fs":        true,
}

func run(pass *analysis.Pass) (interface{}, error) {
	fset := pass.Fset

	for _, file := range pass.Files {
		filename := fset.Position(file.Pos()).Filename
		baseName := filepath.Base(filename)

		if !isBusinessLogicFile(baseName) {
			continue
		}

		for _, imp := range file.Imports {
			path := strings.Trim(imp.Path.Value, `"`)
			if ioPackages[path] {
				pass.Reportf(imp.Pos(), "R18: business logic file %s must not import %q — see docs/ARCHITECTURE.md#R18", baseName, path)
			}
		}
	}

	return nil, nil
}

func isBusinessLogicFile(baseName string) bool {
	name := strings.TrimSuffix(baseName, ".go")
	return name == "service" || name == "types" || strings.HasSuffix(name, "_service")
}
