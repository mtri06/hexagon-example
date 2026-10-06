package dto

import (
	"time"

	"goschool/internal/domain/identity"
)

type User struct {
	ID          int       `json:"id"`
	Username    string    `json:"username"`
	Email       *string   `json:"email"`
	Role        string    `json:"role"`
	Name        string    `json:"name"`
	DateOfBirth Date      `json:"date_of_birth"`
	Gender      string    `json:"gender"`
	IsActive    bool      `json:"is_active"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func NewUser(u *identity.User) User {
	return User{
		ID:          u.ID,
		Username:    u.Username,
		Email:       u.Email,
		Role:        string(u.Role),
		Name:        u.Name,
		DateOfBirth: NewDate(u.DateOfBirth),
		Gender:      string(u.Gender),
		IsActive:    u.IsActive,
		CreatedAt:   u.CreatedAt,
		UpdatedAt:   u.UpdatedAt,
	}
}

// Me is the /auth/me response. Profiles are added with the student and
// teacher aggregates.
type Me struct {
	User User `json:"user"`
}

type LoginRequest struct {
	Username string `json:"username" validate:"required"`
	Password string `json:"password" validate:"required"`
}

type ChangePasswordRequest struct {
	CurrentPassword string `json:"current_password" validate:"required"`
	NewPassword     string `json:"new_password" validate:"required"`
}

type ResetPasswordRequest struct {
	NewPassword string `json:"new_password" validate:"required"`
}

type UserCreateRequest struct {
	Username    string  `json:"username" validate:"required"`
	Password    string  `json:"password" validate:"required"`
	Email       *string `json:"email"`
	Name        string  `json:"name" validate:"required"`
	DateOfBirth Date    `json:"date_of_birth" validate:"required"`
	Gender      string  `json:"gender" validate:"required,oneof=male female other"`
}

func (r UserCreateRequest) ToDomain() identity.CreateUserInput {
	return identity.CreateUserInput{
		Username:    r.Username,
		Password:    r.Password,
		Email:       r.Email,
		Name:        r.Name,
		DateOfBirth: r.DateOfBirth.Time,
		Gender:      identity.Gender(r.Gender),
	}
}

type UserUpdateRequest struct {
	Email       Nullable[string] `json:"email"`
	Name        *string          `json:"name"`
	DateOfBirth *Date            `json:"date_of_birth"`
	Gender      *string          `json:"gender" validate:"omitnil,oneof=male female other"`
	IsActive    *bool            `json:"is_active"`
}

func (r UserUpdateRequest) ToDomain() identity.UpdateUserInput {
	in := identity.UpdateUserInput{
		Email:       r.Email.ToDomain(),
		Name:        r.Name,
		DateOfBirth: r.DateOfBirth.TimePtr(),
		IsActive:    r.IsActive,
	}
	if r.Gender != nil {
		g := identity.Gender(*r.Gender)
		in.Gender = &g
	}
	return in
}
