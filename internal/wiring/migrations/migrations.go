package migrations

import (
	"embed"
	"fmt"
)

//go:embed sql/*.sql
var sqlFiles embed.FS

// AllMigrations is the ordered registry of all database migrations.
// Migrations are executed in slice order — do not reorder.
var AllMigrations = []*Migration{
	{
		Version: 1,
		Name:    "create_metadata_tables",
		UpSQL:   embedSQL("sql/001_create_metadata_tables.up.sql"),
		DownSQL: embedSQL("sql/001_create_metadata_tables.down.sql"),
	},
	{
		Version: 2,
		Name:    "create_audit_tables",
		UpSQL:   embedSQL("sql/002_create_audit_tables.up.sql"),
		DownSQL: embedSQL("sql/002_create_audit_tables.down.sql"),
	},
}

// embedSQL reads an embedded SQL file and returns its contents as a string.
// Panics if the file is not found — this is a compile-time guarantee.
func embedSQL(path string) string {
	data, err := sqlFiles.ReadFile(path)
	if err != nil {
		panic(fmt.Sprintf("failed to embed SQL file %s: %v", path, err))
	}
	return string(data)
}
