package postgres

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/uptrace/bun"
	"github.com/uptrace/bun/driver/pgdriver"

	"goschool/internal/domain/derr"
)

const (
	pgUniqueViolation     = "23505"
	pgForeignKeyViolation = "23503"
	pgCheckViolation      = "23514"
)

// constraintErrors maps a constraint/index name to the domain error a
// violation of it means. Services check these rules first; the mapping is the
// safety net for races.
type constraintErrors map[string]*derr.Error

// translate converts integrity violations into domain errors. Other errors are
// returned unchanged and surface as 500s.
func translate(err error, known constraintErrors) error {
	if err == nil {
		return nil
	}
	var pgErr pgdriver.Error
	if !errors.As(err, &pgErr) || !pgErr.IntegrityViolation() {
		return err
	}
	if de, ok := known[pgErr.Field('n')]; ok {
		return de.Wrap(err)
	}
	switch pgErr.Field('C') {
	case pgUniqueViolation:
		return derr.Conflict("already_exists", "A record with the same values already exists").Wrap(err)
	case pgForeignKeyViolation:
		return derr.Conflict("reference_violation", "The record references, or is referenced by, other records").Wrap(err)
	case pgCheckViolation:
		return derr.Invalid(derr.CodeValidationFailed, "A value is out of the allowed range").Wrap(err)
	}
	return err
}

func isNoRows(err error) bool { return errors.Is(err, sql.ErrNoRows) }

// timestamps is embedded in every row model. bun fills created_at/updated_at
// from the column defaults on insert; updated_at is bumped on every update
// because the schema has no trigger for it.
type timestamps struct {
	CreatedAt time.Time `bun:"created_at,nullzero,notnull,default:current_timestamp"`
	UpdatedAt time.Time `bun:"updated_at,nullzero,notnull,default:current_timestamp"`
}

var _ bun.BeforeAppendModelHook = (*timestamps)(nil)

func (t *timestamps) BeforeAppendModel(_ context.Context, q bun.Query) error {
	if _, ok := q.(*bun.UpdateQuery); ok {
		t.UpdatedAt = time.Now().UTC()
	}
	return nil
}
