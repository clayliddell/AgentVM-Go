package migrations

import (
	"database/sql"
	"io"
	"log/slog"
	"testing"

	_ "modernc.org/sqlite"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// Test helpers
// ---------------------------------------------------------------------------

var testLogger = slog.New(slog.NewTextHandler(io.Discard, nil))

func setupTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func getAppliedVersions(t *testing.T, db *sql.DB) []int {
	t.Helper()
	rows, err := db.Query("SELECT version FROM schema_migrations ORDER BY version")
	require.NoError(t, err)
	defer func() { _ = rows.Close() }()

	var versions []int
	for rows.Next() {
		var v int
		require.NoError(t, rows.Scan(&v))
		versions = append(versions, v)
	}
	return versions
}

func getTables(t *testing.T, db *sql.DB) []string {
	t.Helper()
	rows, err := db.Query("SELECT name FROM sqlite_master WHERE type='table' ORDER BY name")
	require.NoError(t, err)
	defer func() { _ = rows.Close() }()

	var tables []string
	for rows.Next() {
		var name string
		require.NoError(t, rows.Scan(&name))
		tables = append(tables, name)
	}
	return tables
}

// ---------------------------------------------------------------------------
// TestRunner_FreshDatabase
// ---------------------------------------------------------------------------

func TestRunner_FreshDatabase(t *testing.T) {
	db := setupTestDB(t)

	runner, err := NewRunner(db, AllMigrations(), testLogger)
	require.NoError(t, err)

	err = runner.Run()
	require.NoError(t, err)

	// Verify schema_migrations table exists and has all versions.
	versions := getAppliedVersions(t, db)
	assert.Len(t, versions, len(AllMigrations()))
	assert.Equal(t, []int{1, 2}, versions)

	// Verify all expected tables exist.
	tables := getTables(t, db)
	assert.Contains(t, tables, "schema_migrations")
	assert.Contains(t, tables, "wiring_configs")
	assert.Contains(t, tables, "wiring_routes")
	assert.Contains(t, tables, "audit_log")
}

// ---------------------------------------------------------------------------
// TestRunner_AlreadyMigrated
// ---------------------------------------------------------------------------

func TestRunner_AlreadyMigrated(t *testing.T) {
	db := setupTestDB(t)

	runner, err := NewRunner(db, AllMigrations(), testLogger)
	require.NoError(t, err)

	// First run.
	require.NoError(t, runner.Run())

	// Second run (idempotent).
	err = runner.Run()
	require.NoError(t, err)

	// Verify no duplicate versions.
	versions := getAppliedVersions(t, db)
	assert.Len(t, versions, len(AllMigrations()))
	assert.Equal(t, []int{1, 2}, versions)

	// All tables still exist.
	tables := getTables(t, db)
	assert.Contains(t, tables, "schema_migrations")
	assert.Contains(t, tables, "wiring_configs")
	assert.Contains(t, tables, "wiring_routes")
	assert.Contains(t, tables, "audit_log")
}

// ---------------------------------------------------------------------------
// TestRunner_PartiallyMigrated
// ---------------------------------------------------------------------------

func TestRunner_PartiallyMigrated(t *testing.T) {
	db := setupTestDB(t)

	// Manually apply migration 1.
	_, err := db.Exec(AllMigrations()[0].UpSQL)
	require.NoError(t, err)

	// Create schema_migrations and record version 1.
	_, err = db.Exec("CREATE TABLE IF NOT EXISTS schema_migrations (version INTEGER PRIMARY KEY, name TEXT NOT NULL, applied_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP)")
	require.NoError(t, err)
	_, err = db.Exec("INSERT INTO schema_migrations (version, name) VALUES (1, 'create_metadata_tables')")
	require.NoError(t, err)

	// Run full migration.
	runner, err := NewRunner(db, AllMigrations(), testLogger)
	require.NoError(t, err)
	err = runner.Run()
	require.NoError(t, err)

	// Verify both migrations are recorded.
	versions := getAppliedVersions(t, db)
	assert.Equal(t, []int{1, 2}, versions)

	// Verify audit_log table now exists.
	tables := getTables(t, db)
	assert.Contains(t, tables, "audit_log")
}

// ---------------------------------------------------------------------------
// TestRunner_InvalidMigrations
// ---------------------------------------------------------------------------

func TestRunner_NilDB(t *testing.T) {
	_, err := NewRunner(nil, AllMigrations(), testLogger)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "db must not be nil")
}

func TestRunner_EmptyMigrations(t *testing.T) {
	db := setupTestDB(t)
	_, err := NewRunner(db, []*Migration{}, testLogger)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "migrations list must not be empty")
}

func TestRunner_InvalidMigrationVersion(t *testing.T) {
	db := setupTestDB(t)
	invalidMigrations := []*Migration{
		{Version: 0, Name: "bad", UpSQL: "SELECT 1"},
	}
	_, err := NewRunner(db, invalidMigrations, testLogger)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "version must be > 0")
}

func TestRunner_NonSequentialVersions(t *testing.T) {
	db := setupTestDB(t)
	nonSequential := []*Migration{
		{Version: 1, Name: "first", UpSQL: "SELECT 1"},
		{Version: 3, Name: "third", UpSQL: "SELECT 1"},
	}
	_, err := NewRunner(db, nonSequential, testLogger)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "expected version 2")
}

// ---------------------------------------------------------------------------
// TestRunner_DownMigration
// ---------------------------------------------------------------------------

func TestRunner_DownMigration(t *testing.T) {
	db := setupTestDB(t)

	runner, err := NewRunner(db, AllMigrations(), testLogger)
	require.NoError(t, err)

	// Apply all migrations.
	require.NoError(t, runner.Run())

	// Roll back migration 2.
	err = runner.RunDown(2)
	require.NoError(t, err)

	// Verify migration 2 is no longer recorded.
	versions := getAppliedVersions(t, db)
	assert.Equal(t, []int{1}, versions)

	// Verify audit_log table no longer exists.
	tables := getTables(t, db)
	assert.NotContains(t, tables, "audit_log")

	// Verify migration 1 tables still exist.
	assert.Contains(t, tables, "wiring_configs")
	assert.Contains(t, tables, "wiring_routes")
}

func TestRunner_DownMigration_RejectsNonLatestVersion(t *testing.T) {
	db := setupTestDB(t)

	runner, err := NewRunner(db, AllMigrations(), testLogger)
	require.NoError(t, err)
	require.NoError(t, runner.Run())

	err = runner.RunDown(1)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not the latest applied migration")

	versions := getAppliedVersions(t, db)
	assert.Equal(t, []int{1, 2}, versions)
}

func TestRunner_DownMigration_NotFound(t *testing.T) {
	db := setupTestDB(t)

	runner, err := NewRunner(db, AllMigrations(), testLogger)
	require.NoError(t, err)

	err = runner.RunDown(99)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestRunner_DownMigration_NoDownSQL(t *testing.T) {
	db := setupTestDB(t)

	// Create a migration with no DownSQL.
	migrations := []*Migration{
		{Version: 1, Name: "no_down", UpSQL: "CREATE TABLE test_down (id INTEGER PRIMARY KEY)"},
	}
	runner, err := NewRunner(db, migrations, testLogger)
	require.NoError(t, err)

	require.NoError(t, runner.Run())

	err = runner.RunDown(1)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "has no down SQL")
}

// ---------------------------------------------------------------------------
// TestRunner_GetAppliedVersions_Empty
// ---------------------------------------------------------------------------

func TestRunner_GetAppliedVersions_BeforeRun(t *testing.T) {
	db := setupTestDB(t)

	runner, err := NewRunner(db, AllMigrations(), testLogger)
	require.NoError(t, err)

	// Ensure table exists.
	require.NoError(t, runner.EnsureMigrationsTable())

	versions, err := runner.GetAppliedVersions()
	require.NoError(t, err)
	assert.Empty(t, versions)
}
