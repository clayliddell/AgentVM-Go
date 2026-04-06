package migrations

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type captureLogger struct {
	informations []string
}

func (l *captureLogger) Info(msg string, args ...any)  { l.informations = append(l.informations, msg) }
func (l *captureLogger) Warn(msg string, args ...any)  {}
func (l *captureLogger) Error(msg string, args ...any) {}

func TestRunner_LogsAppliedAndSkippedMigrations(t *testing.T) {
	db := setupTestDB(t)
	logger := &captureLogger{}

	runner, err := NewRunner(db, AllMigrations(), logger)
	require.NoError(t, err)
	require.NoError(t, runner.Run())
	require.Len(t, logger.informations, len(AllMigrations()))
	for _, msg := range logger.informations {
		assert.Equal(t, "Applied migration", msg)
	}

	logger.informations = nil
	require.NoError(t, runner.Run())
	require.Len(t, logger.informations, len(AllMigrations()))
	for _, msg := range logger.informations {
		assert.Equal(t, "Skipping migration (already applied)", msg)
	}
}

func TestRunner_ApplyMigrationSkipsWhitespaceStatements(t *testing.T) {
	db := setupTestDB(t)
	logger := &captureLogger{}
	_, err := db.Exec(`CREATE TABLE schema_migrations (
		version INTEGER PRIMARY KEY,
		name TEXT NOT NULL,
		applied_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	)`)
	require.NoError(t, err)

	runner, err := NewRunner(db, []*Migration{{
		Version: 1,
		Name:    "whitespace",
		UpSQL:   "CREATE TABLE demo (id INTEGER PRIMARY KEY);    ; INSERT INTO demo (id) VALUES (1);",
	}}, logger)
	require.NoError(t, err)
	require.NoError(t, runner.applyMigration(runner.migrations[0]))

	var count int
	require.NoError(t, db.QueryRow("SELECT COUNT(*) FROM demo").Scan(&count))
	assert.Equal(t, 1, count)
}

func TestRunner_RunDownSkipsWhitespaceStatements(t *testing.T) {
	db := setupTestDB(t)
	logger := &captureLogger{}

	runner, err := NewRunner(db, []*Migration{{
		Version: 1,
		Name:    "down_whitespace",
		UpSQL:   "CREATE TABLE parent (id INTEGER PRIMARY KEY); CREATE TABLE child (id INTEGER PRIMARY KEY);",
		DownSQL: "DROP TABLE child;    ; DROP TABLE parent;",
	}}, logger)
	require.NoError(t, err)
	require.NoError(t, runner.Run())
	require.NoError(t, runner.RunDown(1))

	tables := getTables(t, db)
	assert.NotContains(t, tables, "parent")
	assert.NotContains(t, tables, "child")
}

func TestRunner_RunDownLogsRollback(t *testing.T) {
	db := setupTestDB(t)
	logger := &captureLogger{}

	runner, err := NewRunner(db, []*Migration{{
		Version: 1,
		Name:    "rollback",
		UpSQL:   "CREATE TABLE rollback (id INTEGER PRIMARY KEY)",
		DownSQL: "DROP TABLE rollback",
	}}, logger)
	require.NoError(t, err)
	require.NoError(t, runner.Run())

	logger.informations = nil
	require.NoError(t, runner.RunDown(1))
	require.Len(t, logger.informations, 1)
	assert.Equal(t, "Rolled back migration", logger.informations[0])
}
