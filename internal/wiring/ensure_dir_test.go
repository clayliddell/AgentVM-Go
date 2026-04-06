package wiring

import (
	"os"
	"path/filepath"
	"testing"
)

func TestEnsureDir(t *testing.T) {
	t.Run("existing directory", func(t *testing.T) {
		dir := t.TempDir()
		if err := ensureDir(dir); err != nil {
			t.Fatalf("ensureDir(%q) returned error: %v", dir, err)
		}
	})

	t.Run("creates missing directory", func(t *testing.T) {
		parent := t.TempDir()
		dir := filepath.Join(parent, "nested")
		if err := ensureDir(dir); err != nil {
			t.Fatalf("ensureDir(%q) returned error: %v", dir, err)
		}
		info, err := os.Stat(dir)
		if err != nil {
			t.Fatalf("expected directory to exist: %v", err)
		}
		if !info.IsDir() {
			t.Fatalf("expected %q to be a directory", dir)
		}
	})

	t.Run("path is file", func(t *testing.T) {
		dir := t.TempDir()
		file := filepath.Join(dir, "file")
		if err := os.WriteFile(file, []byte("x"), 0600); err != nil {
			t.Fatalf("failed to create file: %v", err)
		}
		if err := ensureDir(file); err == nil {
			t.Fatal("expected error when path is a file")
		}
	})
}
