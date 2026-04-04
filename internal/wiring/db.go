package wiring

import (
	"database/sql"
	"fmt"
	"log/slog"

	"agentvm/internal/wiring/migrations"

	_ "modernc.org/sqlite"
)

// InitializeWiringDB opens the SQLite database at the given path, runs all
// pending migrations, and returns the database connection.
//
// If the database is fresh, all migrations are applied.
// If the database is partially migrated, only pending migrations are applied.
// If the database is fully migrated, this returns immediately.
//
// Any migration failure causes the function to return an error and close the
// database connection (fail-fast semantics).
func InitializeWiringDB(dbPath string, logger *slog.Logger) (*sql.DB, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open wiring db: %w", err)
	}

	runner, err := migrations.NewRunner(db, migrations.AllMigrations, logger)
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to create migration runner: %w", err)
	}

	if err := runner.Run(); err != nil {
		db.Close()
		return nil, fmt.Errorf("wiring db migration failed: %w", err)
	}

	return db, nil
}
