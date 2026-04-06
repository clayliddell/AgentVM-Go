package wiring

import (
	"database/sql"
	"io"
	"log/slog"
	"path/filepath"
	"testing"

	_ "modernc.org/sqlite"
)

func TestInitializeWiringDB(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "wiring.db")
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	db, err := InitializeWiringDB(dbPath, logger)
	if err != nil {
		t.Fatalf("InitializeWiringDB returned error: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	if err := requireTable(t, db, "schema_migrations"); err != nil {
		t.Fatal(err)
	}
}

func TestInitializeWiringDBInvalidPath(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "missing", "wiring.db")
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	db, err := InitializeWiringDB(dbPath, logger)
	if err == nil {
		if db != nil {
			_ = db.Close()
		}
		t.Fatal("expected error for invalid database path")
	}
}

func requireTable(t *testing.T, db *sql.DB, name string) error {
	t.Helper()
	var found string
	return db.QueryRow(
		"SELECT name FROM sqlite_master WHERE type='table' AND name = ?",
		name,
	).Scan(&found)
}
