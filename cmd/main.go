package main

import (
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"

	"goschool/internal/db"

	"github.com/joho/godotenv"
	"github.com/rs/zerolog/log"
)

func main() {
	_ = godotenv.Load()

	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "run_server":
		runServer()
	case "migration":
		runMigration()
	case "rollback_migration":
		rollbackMigration()
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n\n", os.Args[1])
		printUsage()
		os.Exit(1)
	}
}

func runServer() {
	addr := getEnv("HTTP_ADDR", ":8080")

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	log.Info().Str("addr", addr).Msg("starting http server")
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatal().Err(err).Msg("http server stopped")
	}
}

func runMigration() {
	client := db.ConnectPostgres(dbConfig())
	defer client.Close()

	db.Migrate(client.DB)
	log.Info().Msg("migration completed")
}

func rollbackMigration() {
	client := db.ConnectPostgres(dbConfig())
	defer client.Close()

	db.RollbackMigration(client.DB)
	log.Info().Msg("rollback migration completed")
}

func dbConfig() db.DBConfig {
	return db.DBConfig{
		Host:        getEnv("DB_HOST", "localhost"),
		Port:        getEnvAsInt("DB_PORT", 5432),
		User:        getEnv("DB_USER", "postgres"),
		Password:    getEnv("DB_PASSWORD", ""),
		Name:        getEnv("DB_NAME", "goschool"),
		SSLMode:     getEnv("DB_SSLMODE", "disable"),
		ConnTimeout: time.Duration(getEnvAsInt("DB_CONN_TIMEOUT_SECONDS", 5)) * time.Second,
	}
}

func getEnv(key string, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}

	return fallback
}

func getEnvAsInt(key string, fallback int) int {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}

	parsed, err := strconv.Atoi(value)
	if err != nil {
		log.Fatal().Err(err).Str("key", key).Str("value", value).Msg("invalid integer environment variable")
	}

	return parsed
}

func printUsage() {
	fmt.Fprintln(os.Stderr, "usage: go run cmd/main.go <command>")
	fmt.Fprintln(os.Stderr)
	fmt.Fprintln(os.Stderr, "commands:")
	fmt.Fprintln(os.Stderr, "  run_server           run http server")
	fmt.Fprintln(os.Stderr, "  migration            run database migrations")
	fmt.Fprintln(os.Stderr, "  rollback_migration   rollback the latest database migration")
}
