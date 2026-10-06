// Package pgtest starts a throwaway Postgres (testcontainers) for integration
// tests. One container is shared by all tests of a package; every call to DB
// truncates all tables, so tests using it must not run in parallel.
package pgtest

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"testing"

	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/uptrace/bun"

	"goschool/internal/adapter/postgres"
	"goschool/internal/db"
)

var (
	once    sync.Once
	sharedD *bun.DB
	initErr error
)

// DB returns a migrated, empty database. Skipped with `go test -short`.
func DB(t testing.TB) *bun.DB {
	t.Helper()
	if testing.Short() {
		t.Skip("integration test: needs Docker")
	}
	once.Do(func() { sharedD, initErr = start() })
	if initErr != nil {
		t.Fatalf("pgtest: %v", initErr)
	}
	if err := truncateAll(context.Background(), sharedD); err != nil {
		t.Fatalf("pgtest: truncate: %v", err)
	}
	return sharedD
}

func start() (*bun.DB, error) {
	ctx := context.Background()
	c, err := tcpostgres.Run(ctx, "postgres:16-alpine",
		tcpostgres.WithDatabase("goschool_test"),
		tcpostgres.WithUsername("test"),
		tcpostgres.WithPassword("test"),
		tcpostgres.BasicWaitStrategies(),
	)
	if err != nil {
		return nil, fmt.Errorf("start container: %w", err)
	}
	// The container is removed by the testcontainers reaper when the test binary exits.
	dsn, err := c.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		return nil, err
	}
	bdb, err := postgres.Open(ctx, dsn, postgres.Options{})
	if err != nil {
		return nil, err
	}
	if err := db.Migrate(bdb.DB); err != nil {
		return nil, err
	}
	return bdb, nil
}

func truncateAll(ctx context.Context, bdb *bun.DB) error {
	var tables []string
	err := bdb.NewSelect().
		Column("tablename").
		Table("pg_tables").
		Where("schemaname = 'public'").
		Where("tablename <> 'goose_db_version'").
		Scan(ctx, &tables)
	if err != nil || len(tables) == 0 {
		return err
	}
	_, err = bdb.ExecContext(ctx, "TRUNCATE "+strings.Join(tables, ", ")+" RESTART IDENTITY CASCADE")
	return err
}
