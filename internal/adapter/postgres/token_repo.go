package postgres

import (
	"context"
	"time"

	"github.com/uptrace/bun"

	"goschool/internal/domain/identity"
)

type tokenRow struct {
	bun.BaseModel `bun:"table:tokens,alias:t"`

	ID            int       `bun:"id,pk,autoincrement"`
	Body          string    `bun:"body,notnull"` // SHA-256 of the secret, never the secret itself
	UserID        int       `bun:"user_id,notnull"`
	Type          string    `bun:"type,notnull"`
	ExpiresAt     time.Time `bun:"expires_at,notnull"`
	IsRevoked     bool      `bun:"is_revoked,notnull"`
	IsUsed        bool      `bun:"is_used,notnull"`
	IsBlacklisted bool      `bun:"is_blacklisted,notnull"`
	timestamps
}

func (r *tokenRow) toDomain() *identity.Token {
	return &identity.Token{
		ID:        r.ID,
		UserID:    r.UserID,
		Hash:      r.Body,
		Type:      identity.TokenType(r.Type),
		ExpiresAt: r.ExpiresAt,
		IsRevoked: r.IsRevoked,
		IsUsed:    r.IsUsed,
		CreatedAt: r.CreatedAt,
	}
}

type TokenRepo struct{ repo }

var _ identity.TokenRepository = TokenRepo{}

func NewTokenRepo(db *bun.DB) TokenRepo { return TokenRepo{repo{db}} }

func (r TokenRepo) Create(ctx context.Context, t *identity.Token) error {
	row := &tokenRow{
		Body:      t.Hash,
		UserID:    t.UserID,
		Type:      string(t.Type),
		ExpiresAt: t.ExpiresAt,
		IsRevoked: t.IsRevoked,
		IsUsed:    t.IsUsed,
	}
	if _, err := r.idb(ctx).NewInsert().Model(row).Returning("id, created_at").Exec(ctx); err != nil {
		return translate(err, nil)
	}
	t.ID, t.CreatedAt = row.ID, row.CreatedAt
	return nil
}

func (r TokenRepo) GetByHashForUpdate(ctx context.Context, hash string, typ identity.TokenType) (*identity.Token, error) {
	row := new(tokenRow)
	err := r.idb(ctx).NewSelect().Model(row).
		Where("t.body = ?", hash).
		Where("t.type = ?", string(typ)).
		For("UPDATE").
		Limit(1).
		Scan(ctx)
	if isNoRows(err) {
		return nil, identity.ErrTokenNotFound
	}
	if err != nil {
		return nil, err
	}
	return row.toDomain(), nil
}

func (r TokenRepo) Update(ctx context.Context, t *identity.Token) error {
	_, err := r.idb(ctx).NewUpdate().Model((*tokenRow)(nil)).
		Set("is_used = ?", t.IsUsed).
		Set("is_revoked = ?", t.IsRevoked).
		Set("updated_at = now()").
		Where("id = ?", t.ID).
		Exec(ctx)
	return err
}

func (r TokenRepo) RevokeAllForUser(ctx context.Context, userID int, typ identity.TokenType) error {
	_, err := r.idb(ctx).NewUpdate().Model((*tokenRow)(nil)).
		Set("is_revoked = TRUE").
		Set("updated_at = now()").
		Where("user_id = ?", userID).
		Where("type = ?", string(typ)).
		Where("is_revoked = FALSE").
		Exec(ctx)
	return err
}
