package db

import (
	"database/sql"
	"embed"
	"fmt"

	"github.com/pressly/goose/v3"
)

//go:embed migration/*.sql
var migrationEmbed embed.FS

func init() {
	goose.SetBaseFS(migrationEmbed)
}

// Migrate applies all pending migrations.
func Migrate(client *sql.DB) error {
	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("set goose dialect: %w", err)
	}
	if err := goose.Up(client, "migration"); err != nil {
		return fmt.Errorf("run migrations: %w", err)
	}
	return nil
}

// RollbackMigration rolls back the most recent migration.
func RollbackMigration(client *sql.DB) error {
	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("set goose dialect: %w", err)
	}
	if err := goose.Down(client, "migration"); err != nil {
		return fmt.Errorf("rollback migration: %w", err)
	}
	return nil
}
