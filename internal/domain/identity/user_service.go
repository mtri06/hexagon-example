package identity

import (
	"context"
	"time"

	"goschool/internal/domain/derr"
	"goschool/internal/domain/port"
	"goschool/internal/domain/shared"
)

type UserServiceDeps struct {
	Users  UserRepository
	Tokens TokenRepository
	Tx     port.TxManager
	Hasher port.PasswordHasher
	Clock  port.Clock
}

// UserService implements account administration across roles. Every use case
// except EnsureAdmin requires an admin actor.
type UserService struct {
	d UserServiceDeps
}

func NewUserService(d UserServiceDeps) *UserService {
	return &UserService{d: d}
}

func (s *UserService) List(ctx context.Context, actor shared.Actor, f UserFilter, p shared.PageRequest) (shared.Page[User], error) {
	if err := actor.RequireAdmin(); err != nil {
		return shared.Page[User]{}, err
	}
	if f.Role != nil {
		if err := f.Role.Validate(); err != nil {
			return shared.Page[User]{}, derr.InvalidField("role", err)
		}
	}
	p = p.Normalize()
	users, total, err := s.d.Users.List(ctx, f, p)
	if err != nil {
		return shared.Page[User]{}, err
	}
	return shared.NewPage(users, p, total), nil
}

func (s *UserService) Get(ctx context.Context, actor shared.Actor, id int) (*User, error) {
	if err := actor.RequireAdmin(); err != nil {
		return nil, err
	}
	return s.d.Users.GetByID(ctx, id)
}

// Update changes common user fields. Email must stay unique, an admin cannot
// deactivate themselves, and deactivation revokes all refresh tokens.
func (s *UserService) Update(ctx context.Context, actor shared.Actor, id int, in UpdateUserInput) (*User, error) {
	if err := actor.RequireAdmin(); err != nil {
		return nil, err
	}
	if id == actor.UserID && in.IsActive != nil && !*in.IsActive {
		return nil, ErrCannotDeactivate
	}

	var u *User
	err := s.d.Tx.WithinTx(ctx, func(ctx context.Context) error {
		var err error
		if u, err = s.d.Users.GetByID(ctx, id); err != nil {
			return err
		}
		prevEmail := u.Email
		deactivated, err := u.ApplyUpdate(in, s.now())
		if err != nil {
			return err
		}
		if u.Email != nil && (prevEmail == nil || *prevEmail != *u.Email) {
			if err := EnsureUnique(ctx, s.d.Users, "", u.Email, u.ID); err != nil {
				return err
			}
		}
		if err := s.d.Users.Update(ctx, u); err != nil {
			return err
		}
		if deactivated {
			return s.d.Tokens.RevokeAllForUser(ctx, u.ID, TokenTypeRefresh)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return u, nil
}

// ResetPassword sets a new password for any user and revokes their sessions.
func (s *UserService) ResetPassword(ctx context.Context, actor shared.Actor, id int, newPassword string) error {
	if err := actor.RequireAdmin(); err != nil {
		return err
	}
	return s.d.Tx.WithinTx(ctx, func(ctx context.Context) error {
		u, err := s.d.Users.GetByID(ctx, id)
		if err != nil {
			return err
		}
		if err := u.SetPassword(newPassword, "new_password", s.d.Hasher); err != nil {
			return err
		}
		if err := s.d.Users.Update(ctx, u); err != nil {
			return err
		}
		return s.d.Tokens.RevokeAllForUser(ctx, u.ID, TokenTypeRefresh)
	})
}

func (s *UserService) CreateAdmin(ctx context.Context, actor shared.Actor, in CreateUserInput) (*User, error) {
	if err := actor.RequireAdmin(); err != nil {
		return nil, err
	}
	return s.create(ctx, in, shared.RoleAdmin)
}

// EnsureAdmin creates the bootstrap admin account if the username is free.
// It is a system operation (CLI), so it takes no actor.
func (s *UserService) EnsureAdmin(ctx context.Context, username, password string) (created bool, err error) {
	exists, err := s.d.Users.UsernameExists(ctx, NormalizeUsername(username))
	if err != nil || exists {
		return false, err
	}
	_, err = s.create(ctx, CreateUserInput{
		Username:    username,
		Password:    password,
		Name:        "Administrator",
		DateOfBirth: shared.NewDate(1970, time.January, 1),
		Gender:      GenderOther,
	}, shared.RoleAdmin)
	return err == nil, err
}

func (s *UserService) create(ctx context.Context, in CreateUserInput, role shared.Role) (*User, error) {
	u, err := NewUser(in, role, s.d.Hasher, s.now())
	if err != nil {
		return nil, err
	}
	if err := EnsureUnique(ctx, s.d.Users, u.Username, u.Email, 0); err != nil {
		return nil, err
	}
	if err := s.d.Users.Create(ctx, u); err != nil {
		return nil, err
	}
	return u, nil
}

func (s *UserService) now() time.Time { return s.d.Clock.Now() }
