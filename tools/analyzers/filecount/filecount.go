package filecount

import (
	"go/token"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/tools/go/analysis"
)

const doc = "R6: flag packages exceeding 10 non-test .go files"

var Analyzer = &analysis.Analyzer{
	Name: "filecount",
	Doc:  doc,
	Run:  run,
}

func run(pass *analysis.Pass) (interface{}, error) {
	const maxFiles = 10

	dir := filepath.Dir(pass.Fset.Position(pass.Files[0].Pos()).Filename)

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	count := 0
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if strings.HasSuffix(name, ".go") && !strings.HasSuffix(name, "_test.go") {
			count++
		}
	}

	if count > maxFiles {
		pass.Reportf(token.NoPos, "R6: package %q has %d non-test .go files, exceeds budget of %d — see docs/ARCHITECTURE.md#R6", pass.Pkg.Path(), count, maxFiles)
	}

	return nil, nil
}
