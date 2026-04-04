// Package migrations provides a versioned SQLite migration runner for the
// wiring layer's metadata and audit schema.
package migrations

import (
	"database/sql"
	"fmt"
	"strings"
)

// Logger is the interface the runner uses for logging.
// *slog.Logger satisfies this interface.
type Logger interface {
	Info(msg string, args ...any)
	Warn(msg string, args ...any)
	Error(msg string, args ...any)
}

// Runner executes pending migrations against a SQLite database.
type Runner struct {
	db         *sql.DB
	migrations []*Migration
	logger     Logger
}

// NewRunner creates a new migration runner.
// It validates that db is non-nil, migrations is non-empty and sorted by version.
func NewRunner(db *sql.DB, migrations []*Migration, logger Logger) (*Runner, error) {
	if db == nil {
		return nil, fmt.Errorf("migrations: db must not be nil")
	}
	if len(migrations) == 0 {
		return nil, fmt.Errorf("migrations: migrations list must not be empty")
	}

	// Validate each migration and ensure sequential ordering.
	for i, m := range migrations {
		if err := m.Validate(); err != nil {
			return nil, fmt.Errorf("migrations: invalid migration at index %d: %w", i, err)
		}
		if i > 0 && m.Version != migrations[i-1].Version+1 {
			return nil, fmt.Errorf(
				"migrations: expected version %d at index %d, got %d",
				migrations[i-1].Version+1, i, m.Version,
			)
		}
	}

	return &Runner{
		db:         db,
		migrations: migrations,
		logger:     logger,
	}, nil
}

// EnsureMigrationsTable creates the schema_migrations table if it does not exist.
// This operation is idempotent.
func (r *Runner) EnsureMigrationsTable() error {
	_, err := r.db.Exec(`
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version INTEGER PRIMARY KEY,
			name TEXT NOT NULL,
			applied_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);
	`)
	if err != nil {
		return fmt.Errorf("failed to create schema_migrations table: %w", err)
	}
	return nil
}

// GetAppliedVersions returns a sorted slice of all applied migration versions.
func (r *Runner) GetAppliedVersions() ([]int, error) {
	rows, err := r.db.Query("SELECT version FROM schema_migrations ORDER BY version")
	if err != nil {
		return nil, fmt.Errorf("failed to query applied migrations: %w", err)
	}
	defer rows.Close()

	var versions []int
	for rows.Next() {
		var v int
		if err := rows.Scan(&v); err != nil {
			return nil, fmt.Errorf("failed to scan migration version: %w", err)
		}
		versions = append(versions, v)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating migration versions: %w", err)
	}
	return versions, nil
}

// Run applies all pending migrations in order.
// It is safe to call multiple times — already-applied migrations are skipped.
func (r *Runner) Run() error {
	if err := r.EnsureMigrationsTable(); err != nil {
		return err
	}

	applied, err := r.GetAppliedVersions()
	if err != nil {
		return err
	}

	// Build a set for O(1) lookup.
	appliedSet := make(map[int]bool, len(applied))
	for _, v := range applied {
		appliedSet[v] = true
	}

	for _, m := range r.migrations {
		if appliedSet[m.Version] {
			r.logger.Info("Skipping migration (already applied)", "version", m.Version, "name", m.Name)
			continue
		}

		if err := r.applyMigration(m); err != nil {
			return fmt.Errorf("migration %d (%s) failed: %w", m.Version, m.Name, err)
		}
		r.logger.Info("Applied migration", "version", m.Version, "name", m.Name)
	}

	return nil
}

// applyMigration executes a single migration within a transaction.
func (r *Runner) applyMigration(m *Migration) error {
	tx, err := r.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	// Execute the migration SQL. Split on semicolons to handle multi-statement SQL.
	statements := splitStatements(m.UpSQL)
	for _, stmt := range statements {
		stmt = strings.TrimSpace(stmt)
		if stmt == "" {
			continue
		}
		if _, err := tx.Exec(stmt); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("failed to execute SQL: %w", err)
		}
	}

	// Record the applied version.
	if _, err := tx.Exec(
		"INSERT INTO schema_migrations (version, name) VALUES (?, ?)",
		m.Version, m.Name,
	); err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("failed to record migration version: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit migration: %w", err)
	}

	return nil
}

// RunDown rolls back a specific migration using its DownSQL.
// Useful for development and testing.
func (r *Runner) RunDown(version int) error {
	var target *Migration
	for _, m := range r.migrations {
		if m.Version == version {
			target = m
			break
		}
	}
	if target == nil {
		return fmt.Errorf("migration version %d not found", version)
	}
	if target.DownSQL == "" {
		return fmt.Errorf("migration %d (%s) has no down SQL", version, target.Name)
	}

	tx, err := r.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	statements := splitStatements(target.DownSQL)
	for _, stmt := range statements {
		stmt = strings.TrimSpace(stmt)
		if stmt == "" {
			continue
		}
		if _, err := tx.Exec(stmt); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("failed to execute down SQL: %w", err)
		}
	}

	if _, err := tx.Exec("DELETE FROM schema_migrations WHERE version = ?", version); err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("failed to remove migration record: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit rollback: %w", err)
	}

	r.logger.Info("Rolled back migration", "version", version, "name", target.Name)
	return nil
}

// splitStatements splits SQL on semicolons, handling multi-statement SQL files.
func splitStatements(sql string) []string {
	return strings.Split(sql, ";")
}
