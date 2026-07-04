package db

import (
	"database/sql"
	"embed"

	"github.com/pressly/goose/v3"
	"github.com/rs/zerolog/log"
)

//go:embed migration/*.sql
var migrationEmbed embed.FS

func Migrate(client *sql.DB) {
	goose.SetBaseFS(migrationEmbed)

	if err := goose.SetDialect("postgres"); err != nil {
		log.Fatal().Err(err).Msg("failed to set dialect")
	}

	if err := goose.Up(client, "migration"); err != nil {
		log.Fatal().Err(err).Msg("failed to run migrations")
	}
}
