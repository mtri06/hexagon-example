package shared

import (
	"errors"

	"goschool/internal/domain/derr"
)

type Role string

const (
	RoleStudent Role = "student"
	RoleTeacher Role = "teacher"
	RoleAdmin   Role = "admin"
)

var errInvalidRole = errors.New("must be one of: student, teacher, admin")

// Validate returns an error whose message describes the allowed values.
func (r Role) Validate() error {
	switch r {
	case RoleStudent, RoleTeacher, RoleAdmin:
		return nil
	}
	return errInvalidRole
}

// Actor is the authenticated caller of a use case.
type Actor struct {
	UserID int
	Role   Role
}

func (a Actor) IsAdmin() bool { return a.Role == RoleAdmin }

// RequireAdmin returns a Forbidden error unless the actor is an admin.
func (a Actor) RequireAdmin() error {
	if !a.IsAdmin() {
		return derr.Forbidden(derr.CodeForbidden, "Only admins can perform this action")
	}
	return nil
}
