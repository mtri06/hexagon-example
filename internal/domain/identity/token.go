package identity

import (
	"crypto/sha256"
	"encoding/hex"
	"time"
)

type TokenType string

const (
	TokenTypeRefresh TokenType = "refresh_token"
)

// Token is a stored opaque token. Only the SHA-256 hash of the secret is
// persisted, so a database leak does not leak usable tokens.
type Token struct {
	ID        int
	UserID    int
	Hash      string
	Type      TokenType
	ExpiresAt time.Time
	IsRevoked bool
	IsUsed    bool
	CreatedAt time.Time
}

func (t *Token) Expired(now time.Time) bool { return !now.Before(t.ExpiresAt) }

// HashSecret returns the hex SHA-256 of a raw token secret.
func HashSecret(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}
