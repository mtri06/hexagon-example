package postgres

import (
	"context"

	"github.com/uptrace/bun"

	"goschool/internal/domain/port"
)

type txKey struct{}

// TxManager implements port.TxManager. The active bun.Tx travels in the
// context so repositories pick it up transparently.
type TxManager struct {
	db *bun.DB
}

var _ port.TxManager = TxManager{}

func NewTxManager(db *bun.DB) TxManager { return TxManager{db: db} }

// WithinTx runs fn in a transaction. Nested calls join the outer transaction.
func (m TxManager) WithinTx(ctx context.Context, fn func(ctx context.Context) error) error {
	if _, ok := ctx.Value(txKey{}).(bun.Tx); ok {
		return fn(ctx)
	}
	return m.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		return fn(context.WithValue(ctx, txKey{}, tx))
	})
}

// repo is embedded by every repository.
type repo struct {
	db *bun.DB
}

// idb returns the transaction in ctx, or the plain DB when there is none.
func (r repo) idb(ctx context.Context) bun.IDB {
	if tx, ok := ctx.Value(txKey{}).(bun.Tx); ok {
		return tx
	}
	return r.db
}
