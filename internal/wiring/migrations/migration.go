package migrations

import (
	"errors"
	"fmt"
)

// Migration represents a single versioned database migration.
type Migration struct {
	Version int    // Sequential version number (1, 2, 3...)
	Name    string // Human-readable name (e.g., "create_metadata_tables")
	UpSQL   string // Forward migration SQL
	DownSQL string // Rollback SQL (optional but recommended)
}

// errInvalidMigration is the sentinel error returned when a migration fails validation.
var errInvalidMigration = errors.New("invalid migration")

// ErrInvalidMigration returns the sentinel error for invalid migrations.
// Use errors.Is(err, migrations.ErrInvalidMigration()) to check.
func ErrInvalidMigration() error {
	return errInvalidMigration
}

// Validate ensures the migration has all required fields.
func (m *Migration) Validate() error {
	if m.Version <= 0 {
		return fmt.Errorf("%w: version must be > 0, got %d", errInvalidMigration, m.Version)
	}
	if m.Name == "" {
		return fmt.Errorf("%w: version %d: name must not be empty", errInvalidMigration, m.Version)
	}
	if m.UpSQL == "" {
		return fmt.Errorf("%w: version %d (%s): up SQL must not be empty", errInvalidMigration, m.Version, m.Name)
	}
	return nil
}

// ParseMigration validates and constructs a Migration.
func ParseMigration(version int, name string, upSQL, downSQL string) (*Migration, error) {
	m := &Migration{
		Version: version,
		Name:    name,
		UpSQL:   upSQL,
		DownSQL: downSQL,
	}
	if err := m.Validate(); err != nil {
		return nil, err
	}
	return m, nil
}
