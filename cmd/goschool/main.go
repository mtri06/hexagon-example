// Command goschool runs the school management service.
//
//	goschool migrate      apply pending migrations
//	goschool rollback     roll back the latest migration
//	goschool seed-admin   create the admin from ADMIN_USERNAME / ADMIN_PASSWORD if missing
//	goschool serve        run the HTTP API
package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/uptrace/bun"

	"goschool/internal/adapter/postgres"
	"goschool/internal/app"
	"goschool/internal/config"
	"goschool/internal/db"
	"goschool/internal/logger"
)

const usage = "usage: goschool <migrate|rollback|seed-admin|serve>"

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, usage)
		os.Exit(2)
	}
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	logger.Init(cfg.IsProd())

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := run(ctx, os.Args[1], cfg); err != nil {
		log.Fatal().Err(err).Str("command", os.Args[1]).Msg("command failed")
	}
}

func run(ctx context.Context, cmd string, cfg config.Config) error {
	switch cmd {
	case "migrate", "rollback", "seed-admin", "serve":
	default:
		return errors.New(usage)
	}

	if err := cfg.ValidateDB(); err != nil {
		return err
	}
	bdb, err := postgres.Open(ctx, postgres.DSN(cfg.DB), postgres.Options{DialTimeout: cfg.DB.DialTimeout})
	if err != nil {
		return err
	}
	defer bdb.Close()

	switch cmd {
	case "migrate":
		return db.Migrate(bdb.DB)
	case "rollback":
		return db.RollbackMigration(bdb.DB)
	case "seed-admin":
		return seedAdmin(ctx, cfg, bdb)
	default:
		return serve(ctx, cfg, bdb)
	}
}

func seedAdmin(ctx context.Context, cfg config.Config, bdb *bun.DB) error {
	if err := cfg.ValidateAdmin(); err != nil {
		return err
	}
	a := app.New(cfg, bdb, app.Overrides{})
	created, err := a.Users.EnsureAdmin(ctx, cfg.Admin.Username, cfg.Admin.Password)
	if err != nil {
		return err
	}
	if created {
		log.Info().Str("username", cfg.Admin.Username).Msg("admin created")
	} else {
		log.Info().Str("username", cfg.Admin.Username).Msg("admin already exists, nothing to do")
	}
	return nil
}

func serve(ctx context.Context, cfg config.Config, bdb *bun.DB) error {
	if err := errors.Join(cfg.ValidateHTTP(), cfg.ValidateJWT()); err != nil {
		return err
	}
	srv := &http.Server{
		Addr:              cfg.HTTP.Addr(),
		Handler:           app.New(cfg, bdb, app.Overrides{}).Handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		log.Info().Str("addr", srv.Addr).Msg("http server listening")
		errCh <- srv.ListenAndServe()
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
	}
	log.Info().Msg("shutting down")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	return srv.Shutdown(shutdownCtx)
}
