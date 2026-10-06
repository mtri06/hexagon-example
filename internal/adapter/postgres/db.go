// Package postgres implements the domain repository ports with bun.
package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"net"
	"net/url"
	"strconv"
	"time"

	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/driver/pgdriver"
	"github.com/uptrace/bun/extra/bundebug"

	"goschool/internal/config"
)

// DSN builds a postgres:// connection string from config.
func DSN(cfg config.DB) string {
	u := url.URL{
		Scheme:   "postgres",
		User:     url.UserPassword(cfg.User, cfg.Password),
		Host:     net.JoinHostPort(cfg.Host, strconv.Itoa(cfg.Port)),
		Path:     "/" + cfg.Name,
		RawQuery: url.Values{"sslmode": {cfg.SSLMode}}.Encode(),
	}
	return u.String()
}

type Options struct {
	DialTimeout time.Duration
}

// Open connects to Postgres, pings it and returns a bun.DB. Every session runs
// with TimeZone=UTC. Set BUNDEBUG=1 (failed queries) or BUNDEBUG=2 (all
// queries) to log SQL.
func Open(ctx context.Context, dsn string, opts Options) (*bun.DB, error) {
	if opts.DialTimeout == 0 {
		opts.DialTimeout = 10 * time.Second
	}
	connector := pgdriver.NewConnector(
		pgdriver.WithDSN(dsn),
		pgdriver.WithDialTimeout(opts.DialTimeout),
		pgdriver.WithConnParams(map[string]any{"TimeZone": "UTC"}),
	)
	sqldb := sql.OpenDB(connector)
	db := bun.NewDB(sqldb, pgdialect.New())
	db.AddQueryHook(bundebug.NewQueryHook(bundebug.FromEnv("BUNDEBUG")))

	pingCtx, cancel := context.WithTimeout(ctx, opts.DialTimeout)
	defer cancel()
	if err := db.PingContext(pingCtx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("ping postgres: %w", err)
	}
	return db, nil
}
