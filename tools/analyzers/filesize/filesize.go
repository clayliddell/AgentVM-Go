package filesize

import (
	"strings"

	"golang.org/x/tools/go/analysis"
)

const doc = "R5: flag files exceeding 500 lines (test files excluded)"

var Analyzer = &analysis.Analyzer{
	Name: "filesize",
	Doc:  doc,
	Run:  run,
}

func run(pass *analysis.Pass) (interface{}, error) {
	const maxLines = 500

	for _, file := range pass.Files {
		fset := pass.Fset
		filename := fset.Position(file.Pos()).Filename

		if strings.HasSuffix(filename, "_test.go") {
			continue
		}

		lastLine := fset.Position(file.End()).Line
		if lastLine > maxLines {
			pass.Reportf(file.Pos(), "R5: file %q has %d lines, exceeds budget of %d — see docs/ARCHITECTURE.md#R5", filename, lastLine, maxLines)
		}
	}

	return nil, nil
}
