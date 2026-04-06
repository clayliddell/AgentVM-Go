package mutablestate

import (
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"golang.org/x/tools/go/analysis"
)

func TestRunReportsPackageLevelStateAndSkipsExemptions(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "internal", "state")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	sources := map[string]string{
		"state.go": `package state

import "embed"

const constValue = 1

var counter int
var Analyzer = 1
var errThing = 2
var content embed.FS
var regular = 3
`,
		"state_test.go": `package state

var testValue = 4
`,
	}

	names := make([]string, 0, len(sources))
	for name := range sources {
		names = append(names, name)
	}
	sort.Strings(names)

	fset := token.NewFileSet()
	files := make([]*ast.File, 0, len(names))
	for _, name := range names {
		path := filepath.Join(dir, name)
		if err := os.WriteFile(path, []byte(sources[name]), 0o644); err != nil {
			t.Fatalf("write %s: %v", path, err)
		}
		file, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
		if err != nil {
			t.Fatalf("parse %s: %v", path, err)
		}
		files = append(files, file)
	}

	msgs := []string{}
	pass := &analysis.Pass{
		Fset:  fset,
		Files: files,
		Pkg:   types.NewPackage("example.com/project/internal/state", "state"),
		Report: func(d analysis.Diagnostic) {
			msgs = append(msgs, d.Message)
		},
	}

	if _, err := run(pass); err != nil {
		t.Fatalf("run returned error: %v", err)
	}

	if len(msgs) != 2 {
		t.Fatalf("expected 2 diagnostics for package-level mutable state, got %d: %v", len(msgs), msgs)
	}
}
