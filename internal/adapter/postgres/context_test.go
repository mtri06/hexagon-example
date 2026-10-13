package postgres_test

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"goschool/internal/adapter/postgres"
	"goschool/internal/adapter/postgres/pgtest"
)

// respond.Error classifies canceled and timed-out requests. These tests pin
// down how pgdriver reports them, so a driver upgrade that changes it fails
// here instead of silently turning into 500s.
func TestDriverContextErrors(t *testing.T) {
	db := pgtest.DB(t)
	users := postgres.NewUserRepo(db)

	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := users.GetByID(canceled, 1); !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled before query: got %v", err)
	}

	// A deadline mid-query surfaces as a socket timeout, not as
	// context.DeadlineExceeded.
	timeout, cancelTimeout := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancelTimeout()
	_, err := db.ExecContext(timeout, "SELECT pg_sleep(5)")
	if !errors.Is(err, os.ErrDeadlineExceeded) {
		t.Fatalf("deadline mid-query: got %v (%T)", err, err)
	}

	// The pool recovers: the next query works.
	var n int
	if err := db.QueryRowContext(context.Background(), "SELECT 1").Scan(&n); err != nil || n != 1 {
		t.Fatalf("query after timeout: %v", err)
	}
}
