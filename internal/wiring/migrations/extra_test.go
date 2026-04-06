package migrations

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMigrationValidateAndParse(t *testing.T) {
	valid, err := ParseMigration(3, "create_things", "CREATE TABLE things (id INTEGER PRIMARY KEY)", "DROP TABLE things")
	require.NoError(t, err)
	assert.Equal(t, 3, valid.Version)
	assert.Equal(t, "create_things", valid.Name)

	_, err = ParseMigration(0, "bad", "SELECT 1", "")
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrInvalidMigration()))

	for _, tc := range []Migration{
		{Version: 1, Name: "", UpSQL: "SELECT 1"},
		{Version: 1, Name: "bad", UpSQL: ""},
	} {
		tc := tc
		t.Run(tc.Name, func(t *testing.T) {
			err := tc.Validate()
			require.Error(t, err)
			assert.True(t, errors.Is(err, ErrInvalidMigration()))
		})
	}
}

func TestAllMigrationsReturnsCopy(t *testing.T) {
	first := AllMigrations()
	require.Len(t, first, 2)
	first[0] = nil

	second := AllMigrations()
	require.Len(t, second, 2)
	require.NotNil(t, second[0])
	assert.Equal(t, 1, second[0].Version)
}

func TestEmbedSQLMissingFilePanics(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("expected embedSQL to panic for a missing file")
		}
	}()

	embedSQL("sql/does-not-exist.sql")
}

func TestRunner_RunInvalidSQLRollsBack(t *testing.T) {
	db := setupTestDB(t)
	runner, err := NewRunner(db, []*Migration{{
		Version: 1,
		Name:    "broken_migration",
		UpSQL:   "CREATE TABLE broken (id INTEGER PRIMARY KEY); INSERT INTO missing_table VALUES (1)",
	}}, testLogger)
	require.NoError(t, err)

	err = runner.Run()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to execute SQL")

	versions := getAppliedVersions(t, db)
	assert.Empty(t, versions)

	tables := getTables(t, db)
	assert.Contains(t, tables, "schema_migrations")
	assert.NotContains(t, tables, "broken")
}

func TestRunner_TableHelpersErrorOnClosedDB(t *testing.T) {
	db := setupTestDB(t)
	runner, err := NewRunner(db, AllMigrations(), testLogger)
	require.NoError(t, err)

	require.NoError(t, db.Close())

	err = runner.EnsureMigrationsTable()
	require.Error(t, err)

	_, err = runner.GetAppliedVersions()
	require.Error(t, err)
}

func TestRunner_ApplyMigrationBeginError(t *testing.T) {
	db := setupTestDB(t)
	runner, err := NewRunner(db, []*Migration{{
		Version: 1,
		Name:    "begin_error",
		UpSQL:   "SELECT 1",
	}}, testLogger)
	require.NoError(t, err)

	require.NoError(t, db.Close())

	err = runner.applyMigration(runner.migrations[0])
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to begin transaction")
}

func TestRunner_ApplyMigrationRecordError(t *testing.T) {
	db := setupTestDB(t)
	_, err := db.Exec(`CREATE TABLE schema_migrations (
		version INTEGER PRIMARY KEY,
		name TEXT NOT NULL,
		applied_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	)`)
	require.NoError(t, err)
	_, err = db.Exec(`CREATE TRIGGER block_schema_migrations_insert
		BEFORE INSERT ON schema_migrations
		BEGIN
			SELECT RAISE(ABORT, 'insert blocked');
		END;`)
	require.NoError(t, err)

	runner, err := NewRunner(db, []*Migration{{
		Version: 1,
		Name:    "record_error",
		UpSQL:   "CREATE TABLE record_error (id INTEGER PRIMARY KEY)",
	}}, testLogger)
	require.NoError(t, err)

	err = runner.applyMigration(runner.migrations[0])
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to record migration version")
}

func TestRunner_GetAppliedVersionsScanError(t *testing.T) {
	db := setupTestDB(t)
	_, err := db.Exec(`CREATE TABLE schema_migrations (version TEXT)`)
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO schema_migrations (version) VALUES (NULL)`)
	require.NoError(t, err)

	runner, err := NewRunner(db, AllMigrations(), testLogger)
	require.NoError(t, err)

	_, err = runner.GetAppliedVersions()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "scan migration version")
}

func TestRunner_RunDownBeginError(t *testing.T) {
	db := setupTestDB(t)
	runner, err := NewRunner(db, []*Migration{{
		Version: 1,
		Name:    "down_begin_error",
		UpSQL:   "CREATE TABLE down_begin_error (id INTEGER PRIMARY KEY)",
		DownSQL: "DROP TABLE down_begin_error",
	}}, testLogger)
	require.NoError(t, err)

	require.NoError(t, db.Close())

	err = runner.RunDown(1)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to query applied migrations")
}

func TestRunner_RunDownDeleteError(t *testing.T) {
	db := setupTestDB(t)
	runner, err := NewRunner(db, []*Migration{{
		Version: 1,
		Name:    "down_delete_error",
		UpSQL:   "CREATE TABLE down_delete_error (id INTEGER PRIMARY KEY)",
		DownSQL: "DROP TABLE down_delete_error",
	}}, testLogger)
	require.NoError(t, err)

	require.NoError(t, runner.Run())
	_, err = db.Exec(`CREATE TRIGGER block_schema_migrations_delete
		BEFORE DELETE ON schema_migrations
		BEGIN
			SELECT RAISE(ABORT, 'delete blocked');
		END;`)
	require.NoError(t, err)

	err = runner.RunDown(1)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to remove migration record")

	tables := getTables(t, db)
	assert.Contains(t, tables, "down_delete_error")
}

func TestSplitStatements_QuotedSemicolons(t *testing.T) {
	stmts := splitStatements("CREATE TABLE demo (note TEXT DEFAULT 'a;b'); INSERT INTO demo (note) VALUES (\"x;y\");")
	require.Equal(t, []string{
		"CREATE TABLE demo (note TEXT DEFAULT 'a;b')",
		"INSERT INTO demo (note) VALUES (\"x;y\")",
	}, stmts)
}
