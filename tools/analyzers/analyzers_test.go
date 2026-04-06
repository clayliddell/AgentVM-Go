package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"analyzers/baninit"
	"analyzers/filecount"
	"analyzers/filesize"
	"analyzers/importlocation"
	"analyzers/ioseparation"
	"analyzers/sharedtypes"
	"analyzers/wiringonly"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

func runAnalyzer(t *testing.T, analyzer *analysis.Analyzer, pkgPath, relDir string, sources map[string]string) []string {
	t.Helper()

	root := t.TempDir()
	dir := root
	if relDir != "" {
		dir = filepath.Join(root, relDir)
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", dir, err)
	}

	names := make([]string, 0, len(sources))
	for name := range sources {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		path := filepath.Join(dir, name)
		if err := os.WriteFile(path, []byte(sources[name]), 0o644); err != nil {
			t.Fatalf("write %s: %v", path, err)
		}
	}

	fset := token.NewFileSet()
	files := make([]*ast.File, 0, len(names))
	for _, name := range names {
		path := filepath.Join(dir, name)
		file, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
		if err != nil {
			t.Fatalf("parse %s: %v", path, err)
		}
		files = append(files, file)
	}
	if len(files) == 0 {
		t.Fatal("no source files parsed")
	}

	msgs := []string{}
	pass := &analysis.Pass{
		Analyzer: analyzer,
		Fset:     fset,
		Files:    files,
		Pkg:      types.NewPackage(pkgPath, files[0].Name.Name),
		Report: func(d analysis.Diagnostic) {
			msgs = append(msgs, d.Message)
		},
		ResultOf: map[*analysis.Analyzer]any{
			inspect.Analyzer: inspector.New(files),
		},
	}

	if _, err := analyzer.Run(pass); err != nil {
		t.Fatalf("analyzer %s returned error: %v", analyzer.Name, err)
	}

	return msgs
}

func makeLineFile(totalLines int) string {
	var b strings.Builder
	b.WriteString("package p\n")
	b.WriteString("\nvar (\n")
	for i := 0; i < totalLines-4; i++ {
		fmt.Fprintf(&b, "\t_ = %d\n", i)
	}
	b.WriteString(")\n")
	return b.String()
}

func TestBanInitAnalyzer(t *testing.T) {
	t.Run("internal package reports package init only", func(t *testing.T) {
		msgs := runAnalyzer(t, baninit.Analyzer, "example.com/project/internal/foo", "internal/foo", map[string]string{
			"init.go": `package foo

func init() {}

type S struct{}

func (s S) init() {}
`,
			"init_test.go": `package foo

func init() {}
`,
		})

		if len(msgs) != 1 {
			t.Fatalf("expected 1 diagnostic, got %d: %v", len(msgs), msgs)
		}
	})

	t.Run("external package ignored", func(t *testing.T) {
		msgs := runAnalyzer(t, baninit.Analyzer, "example.com/project/foo", "foo", map[string]string{
			"init.go": `package foo

func init() {}
`,
		})

		if len(msgs) != 0 {
			t.Fatalf("expected no diagnostics outside internal/, got %v", msgs)
		}
	})
}

func TestFileCountAnalyzer(t *testing.T) {
	t.Run("exactly ten non-test files passes", func(t *testing.T) {
		sources := map[string]string{}
		for i := 0; i < 10; i++ {
			sources[fmt.Sprintf("f%02d.go", i)] = "package p\n"
		}
		sources["f_test.go"] = "package p\n"

		msgs := runAnalyzer(t, filecount.Analyzer, "example.com/project/p", "pkg", sources)
		if len(msgs) != 0 {
			t.Fatalf("expected no diagnostics at file budget, got %v", msgs)
		}
	})

	t.Run("eleven non-test files reports", func(t *testing.T) {
		sources := map[string]string{}
		for i := 0; i < 11; i++ {
			sources[fmt.Sprintf("f%02d.go", i)] = "package p\n"
		}
		sources["f_test.go"] = "package p\n"

		msgs := runAnalyzer(t, filecount.Analyzer, "example.com/project/p", "pkg", sources)
		if len(msgs) != 1 {
			t.Fatalf("expected one filecount diagnostic, got %d: %v", len(msgs), msgs)
		}
	})
}

func TestFileSizeAnalyzer(t *testing.T) {
	t.Run("test file over limit is ignored", func(t *testing.T) {
		msgs := runAnalyzer(t, filesize.Analyzer, "example.com/project/p", "pkg", map[string]string{
			"small.go":    makeLineFile(500),
			"big_test.go": makeLineFile(501),
		})
		if len(msgs) != 0 {
			t.Fatalf("expected test files to be ignored, got %v", msgs)
		}
	})

	t.Run("regular file over limit reports", func(t *testing.T) {
		msgs := runAnalyzer(t, filesize.Analyzer, "example.com/project/p", "pkg", map[string]string{
			"big.go": makeLineFile(501),
		})
		if len(msgs) != 1 {
			t.Fatalf("expected one filesize diagnostic, got %d: %v", len(msgs), msgs)
		}
	})
}

func TestImportLocationAnalyzer(t *testing.T) {
	msgs := runAnalyzer(t, importlocation.Analyzer, "example.com/project/internal/store", "store", map[string]string{
		"store.go": `package store

import "database/sql"

var _ = sql.ErrNoRows
`,
		"handler.go": `package store

import "net/http"

var _ = http.MethodGet
`,
		"other.go": `package store

import (
	"database/sql"
	"net/http"
)

var _, _ = sql.ErrNoRows, http.MethodGet
`,
		"other_test.go": `package store

import (
	"database/sql"
	"net/http"
)

var _, _ = sql.ErrNoRows, http.MethodGet
`,
	})

	if len(msgs) != 2 {
		t.Fatalf("expected two import-location diagnostics, got %d: %v", len(msgs), msgs)
	}
}

func TestIOSeparationAnalyzer(t *testing.T) {
	msgs := runAnalyzer(t, ioseparation.Analyzer, "example.com/project/internal/service", "service", map[string]string{
		"service.go": `package service

import "os"

var _ = os.PathSeparator
`,
		"types.go": `package service

import "io"

var _ = io.EOF
`,
		"worker_service.go": `package service

import "io/fs"

var _ = fs.FileMode(0)
`,
		"handler.go": `package service

import "database/sql"

var _ = sql.ErrNoRows
`,
	})

	if len(msgs) != 3 {
		t.Fatalf("expected three IO separation diagnostics, got %d: %v", len(msgs), msgs)
	}
}

func TestSharedTypesAnalyzer(t *testing.T) {
	msgs := runAnalyzer(t, sharedtypes.Analyzer, "example.com/project/shared/types", "types", map[string]string{
		"types.go": `package types

import (
	"example.com/project/features/foo"
	"fmt"
)

var _ = fmt.Sprintf
var _ = foo.Name
`,
	})

	if len(msgs) != 1 {
		t.Fatalf("expected one shared/types diagnostic, got %d: %v", len(msgs), msgs)
	}
}

func TestWiringOnlyAnalyzer(t *testing.T) {
	msgs := runAnalyzer(t, wiringonly.Analyzer, "example.com/project/internal/app", "app", map[string]string{
		"app.go": `package app

import (
	_ "example.com/project/internal/features/alpha"
	_ "example.com/project/internal/features/beta"
)
`,
	})

	if len(msgs) != 1 {
		t.Fatalf("expected one wiring-only diagnostic, got %d: %v", len(msgs), msgs)
	}
}
