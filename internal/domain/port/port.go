// Package port declares driven ports shared by several aggregates. Adapters
// (postgres, security) implement them; services depend only on these interfaces.
package port

import (
	"context"
	"time"

	"goschool/internal/domain/shared"
)

// TxManager runs fn inside a database transaction. Repositories called with the
// ctx passed to fn take part in that transaction. If fn returns an error the
// transaction is rolled back and that error is returned unchanged.
type TxManager interface {
	WithinTx(ctx context.Context, fn func(ctx context.Context) error) error
}

type PasswordHasher interface {
	Hash(password string) (string, error)
	// Matches reports whether password matches hash.
	Matches(hash, password string) bool
}

// TokenIssuer signs short-lived access tokens.
type TokenIssuer interface {
	IssueAccessToken(actor shared.Actor) (token string, expiresAt time.Time, err error)
}

// SecretGenerator produces unguessable opaque strings (e.g. refresh tokens).
type SecretGenerator interface {
	NewSecret() (string, error)
}

type Clock interface {
	Now() time.Time
}

// SystemClock is the real-time Clock.
type SystemClock struct{}

func (SystemClock) Now() time.Time { return time.Now().UTC() }
