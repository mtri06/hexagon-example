package identity

import (
	"context"
	"time"

	"goschool/internal/domain/derr"
	"goschool/internal/domain/port"
	"goschool/internal/domain/shared"
)

type TokenPair struct {
	AccessToken      string
	AccessExpiresAt  time.Time
	RefreshToken     string
	RefreshExpiresAt time.Time
}

type LoginResult struct {
	User   *User
	Tokens TokenPair
}

type AuthServiceDeps struct {
	Users      UserRepository
	Tokens     TokenRepository
	Tx         port.TxManager
	Hasher     port.PasswordHasher
	Issuer     port.TokenIssuer
	Secrets    port.SecretGenerator
	Clock      port.Clock
	RefreshTTL time.Duration
}

// AuthService implements login, refresh-token rotation, logout and
// self-service password changes.
type AuthService struct {
	d AuthServiceDeps
}

func NewAuthService(d AuthServiceDeps) *AuthService {
	return &AuthService{d: d}
}

// Login checks credentials and issues a token pair. The same error is
// returned for an unknown username and a wrong password.
func (s *AuthService) Login(ctx context.Context, username, password string) (*LoginResult, error) {
	u, err := s.d.Users.GetByUsername(ctx, NormalizeUsername(username))
	if derr.Is(err, CodeUserNotFound) {
		return nil, ErrInvalidCredentials
	}
	if err != nil {
		return nil, err
	}
	if !u.PasswordMatches(password, s.d.Hasher) {
		return nil, ErrInvalidCredentials
	}
	if !u.IsActive {
		return nil, ErrUserInactive
	}
	pair, err := s.issue(ctx, u)
	if err != nil {
		return nil, err
	}
	return &LoginResult{User: u, Tokens: pair}, nil
}

// Refresh rotates a refresh token: the presented token is marked used and a
// new pair is issued. Presenting an already-used token revokes every refresh
// token of its owner (likely theft) and fails.
func (s *AuthService) Refresh(ctx context.Context, rawRefreshToken string) (TokenPair, error) {
	if rawRefreshToken == "" {
		return TokenPair{}, ErrInvalidRefreshToken
	}
	var (
		pair   TokenPair
		reused bool
	)
	err := s.d.Tx.WithinTx(ctx, func(ctx context.Context) error {
		tok, err := s.d.Tokens.GetByHashForUpdate(ctx, HashSecret(rawRefreshToken), TokenTypeRefresh)
		if derr.Is(err, CodeTokenNotFound) {
			return ErrInvalidRefreshToken
		}
		if err != nil {
			return err
		}
		if tok.IsUsed {
			// Commit the revocation, then report the reuse outside the tx.
			reused = true
			return s.d.Tokens.RevokeAllForUser(ctx, tok.UserID, TokenTypeRefresh)
		}
		if tok.IsRevoked {
			return ErrInvalidRefreshToken
		}
		if tok.Expired(s.d.Clock.Now()) {
			return ErrRefreshTokenExpired
		}

		u, err := s.d.Users.GetByID(ctx, tok.UserID)
		if err != nil {
			return err
		}
		if !u.IsActive {
			return ErrUserInactive
		}

		tok.IsUsed = true
		if err := s.d.Tokens.Update(ctx, tok); err != nil {
			return err
		}
		pair, err = s.issue(ctx, u)
		return err
	})
	if err != nil {
		return TokenPair{}, err
	}
	if reused {
		return TokenPair{}, ErrRefreshTokenReused
	}
	return pair, nil
}

// Logout revokes the given refresh token if it belongs to the actor. Unknown
// or foreign tokens are ignored so logout always succeeds.
func (s *AuthService) Logout(ctx context.Context, actor shared.Actor, rawRefreshToken string) error {
	if rawRefreshToken == "" {
		return nil
	}
	return s.d.Tx.WithinTx(ctx, func(ctx context.Context) error {
		tok, err := s.d.Tokens.GetByHashForUpdate(ctx, HashSecret(rawRefreshToken), TokenTypeRefresh)
		if derr.Is(err, CodeTokenNotFound) {
			return nil
		}
		if err != nil {
			return err
		}
		if tok.UserID != actor.UserID || tok.IsRevoked {
			return nil
		}
		tok.IsRevoked = true
		return s.d.Tokens.Update(ctx, tok)
	})
}

// Me returns the actor's own account.
func (s *AuthService) Me(ctx context.Context, actor shared.Actor) (*User, error) {
	return s.d.Users.GetByID(ctx, actor.UserID)
}

// ChangePassword verifies the current password, sets the new one and revokes
// every refresh token of the user.
func (s *AuthService) ChangePassword(ctx context.Context, actor shared.Actor, current, next string) error {
	return s.d.Tx.WithinTx(ctx, func(ctx context.Context) error {
		u, err := s.d.Users.GetByID(ctx, actor.UserID)
		if err != nil {
			return err
		}
		if !u.PasswordMatches(current, s.d.Hasher) {
			return derr.Invalid(CodeCurrentPasswordIncorrect, "Current password is incorrect",
				derr.FieldError{Field: "current_password", Message: "is incorrect"})
		}
		if current == next {
			return derr.Invalid(derr.CodeValidationFailed, "Invalid input",
				derr.FieldError{Field: "new_password", Message: "must differ from the current password"})
		}
		if err := u.SetPassword(next, "new_password", s.d.Hasher); err != nil {
			return err
		}
		if err := s.d.Users.Update(ctx, u); err != nil {
			return err
		}
		return s.d.Tokens.RevokeAllForUser(ctx, u.ID, TokenTypeRefresh)
	})
}

func (s *AuthService) issue(ctx context.Context, u *User) (TokenPair, error) {
	access, accessExp, err := s.d.Issuer.IssueAccessToken(u.Actor())
	if err != nil {
		return TokenPair{}, err
	}
	raw, err := s.d.Secrets.NewSecret()
	if err != nil {
		return TokenPair{}, err
	}
	tok := &Token{
		UserID:    u.ID,
		Hash:      HashSecret(raw),
		Type:      TokenTypeRefresh,
		ExpiresAt: s.d.Clock.Now().Add(s.d.RefreshTTL),
	}
	if err := s.d.Tokens.Create(ctx, tok); err != nil {
		return TokenPair{}, err
	}
	return TokenPair{
		AccessToken:      access,
		AccessExpiresAt:  accessExp,
		RefreshToken:     raw,
		RefreshExpiresAt: tok.ExpiresAt,
	}, nil
}
