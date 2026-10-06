package identity

import (
	"time"

	"goschool/internal/domain/shared"
)

type CreateUserInput struct {
	Username    string
	Password    string
	Email       *string
	Name        string
	DateOfBirth time.Time
	Gender      Gender
}

// UpdateUserInput is a partial update; nil / unset fields are left unchanged.
type UpdateUserInput struct {
	Email       shared.Nullable[string]
	Name        *string
	DateOfBirth *time.Time
	Gender      *Gender
	IsActive    *bool
}

func (in UpdateUserInput) IsEmpty() bool {
	return !in.Email.Set && in.Name == nil && in.DateOfBirth == nil && in.Gender == nil && in.IsActive == nil
}

type UserFilter struct {
	Role     *shared.Role
	IsActive *bool
	// Query matches username, name or email (case-insensitive, substring).
	Query *string
}
