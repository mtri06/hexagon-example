package identity

import (
	"context"

	"goschool/internal/domain/shared"
)

type UserRepository interface {
	// Create inserts u and sets its ID and timestamps.
	Create(ctx context.Context, u *User) error
	// Update persists all mutable fields of u and refreshes UpdatedAt.
	Update(ctx context.Context, u *User) error
	// GetByID returns ErrUserNotFound when missing.
	GetByID(ctx context.Context, id int) (*User, error)
	// GetByUsername returns ErrUserNotFound when missing.
	GetByUsername(ctx context.Context, username string) (*User, error)
	UsernameExists(ctx context.Context, username string) (bool, error)
	// EmailExists ignores the user with excludeUserID (0 to ignore none).
	EmailExists(ctx context.Context, email string, excludeUserID int) (bool, error)
	List(ctx context.Context, f UserFilter, p shared.PageRequest) ([]User, int, error)
}

type TokenRepository interface {
	// Create inserts t and sets its ID.
	Create(ctx context.Context, t *Token) error
	// GetByHashForUpdate returns ErrTokenNotFound when missing and locks the
	// row when called inside a transaction.
	GetByHashForUpdate(ctx context.Context, hash string, typ TokenType) (*Token, error)
	// Update persists IsUsed and IsRevoked.
	Update(ctx context.Context, t *Token) error
	RevokeAllForUser(ctx context.Context, userID int, typ TokenType) error
}

// EnsureUnique checks that username (when non-empty) and email (when non-nil)
// are not used by another user. Aggregates that create users (student,
// teacher) call it before inserting.
func EnsureUnique(ctx context.Context, users UserRepository, username string, email *string, excludeUserID int) error {
	if username != "" {
		exists, err := users.UsernameExists(ctx, username)
		if err != nil {
			return err
		}
		if exists {
			return ErrUsernameTaken
		}
	}
	if email != nil {
		exists, err := users.EmailExists(ctx, *email, excludeUserID)
		if err != nil {
			return err
		}
		if exists {
			return ErrEmailTaken
		}
	}
	return nil
}
