package main

import (
	"database/sql"
	"embed"
	"fmt"
	"io/fs"
)

//go:embed migrations/*.sql
var migrationAssets embed.FS

// applyEmbeddedMigrations preserves the original incremental, idempotent SQL order.
func applyEmbeddedMigrations(db *sql.DB) error {
	names, err := fs.Glob(migrationAssets, "migrations/*.sql")
	if err != nil {
		return err
	}
	for _, name := range names {
		statement, err := migrationAssets.ReadFile(name)
		if err != nil {
			return fmt.Errorf("read %s: %w", name, err)
		}
		if _, err := db.Exec(string(statement)); err != nil {
			return fmt.Errorf("apply %s: %w", name, err)
		}
	}
	return nil
}
