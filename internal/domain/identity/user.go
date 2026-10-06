// Package identity owns user accounts and authentication.
package identity

import (
	"errors"
	"net/mail"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"goschool/internal/domain/derr"
	"goschool/internal/domain/port"
	"goschool/internal/domain/shared"
)

type Gender string

const (
	GenderMale   Gender = "male"
	GenderFemale Gender = "female"
	GenderOther  Gender = "other"
)

var errInvalidGender = errors.New("must be one of: male, female, other")

// Validate returns an error whose message describes the allowed values.
func (g Gender) Validate() error {
	switch g {
	case GenderMale, GenderFemale, GenderOther:
		return nil
	}
	return errInvalidGender
}

const (
	minPasswordLen = 8
	maxPasswordLen = 72 // bcrypt ignores bytes beyond 72
	maxEmailLen    = 100
	maxNameLen     = 255
)

var (
	usernamePattern = regexp.MustCompile(`^[a-z0-9._-]{3,100}$`)
	minDateOfBirth  = shared.NewDate(1900, time.January, 1)
)

// User is an account of any role. Role-specific data lives in the student and
// teacher aggregates, which embed a User.
type User struct {
	ID           int
	Username     string
	PasswordHash string
	Email        *string
	Role         shared.Role
	Name         string
	DateOfBirth  time.Time
	Gender       Gender
	IsActive     bool
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

func (u *User) Actor() shared.Actor { return shared.Actor{UserID: u.ID, Role: u.Role} }

// NewUser validates the input and builds an active user with a hashed
// password. now is used to check that the date of birth is in the past.
func NewUser(in CreateUserInput, role shared.Role, hasher port.PasswordHasher, now time.Time) (*User, error) {
	u := &User{
		Username:    NormalizeUsername(in.Username),
		Email:       normalizeEmail(in.Email),
		Role:        role,
		Name:        strings.TrimSpace(in.Name),
		DateOfBirth: shared.DateOf(in.DateOfBirth),
		Gender:      in.Gender,
		IsActive:    true,
	}

	f := derr.FieldErrors{}
	validateUsername(f, u.Username)
	validatePassword(f, "password", in.Password)
	f.AddErr("role", u.Role.Validate())
	validateEmail(f, u.Email)
	validateName(f, u.Name)
	validateDateOfBirth(f, u.DateOfBirth, now)
	f.AddErr("gender", u.Gender.Validate())
	if err := f.Err(); err != nil {
		return nil, err
	}

	hash, err := hasher.Hash(in.Password)
	if err != nil {
		return nil, err
	}
	u.PasswordHash = hash
	return u, nil
}

// ApplyUpdate validates and applies a partial update. Only the provided
// fields are validated; username and role are immutable. The user is left
// untouched if any field is invalid. It reports whether the user was
// deactivated by this update.
func (u *User) ApplyUpdate(in UpdateUserInput, now time.Time) (deactivated bool, err error) {
	if in.IsEmpty() {
		return false, ErrNothingToUpdate
	}
	next := *u
	f := derr.FieldErrors{}
	if in.Email.Set {
		next.Email = normalizeEmail(in.Email.Value)
		validateEmail(f, next.Email)
	}
	if in.Name != nil {
		next.Name = strings.TrimSpace(*in.Name)
		validateName(f, next.Name)
	}
	if in.DateOfBirth != nil {
		next.DateOfBirth = shared.DateOf(*in.DateOfBirth)
		validateDateOfBirth(f, next.DateOfBirth, now)
	}
	if in.Gender != nil {
		next.Gender = *in.Gender
		f.AddErr("gender", next.Gender.Validate())
	}
	if in.IsActive != nil {
		next.IsActive = *in.IsActive
	}
	if err := f.Err(); err != nil {
		return false, err
	}
	deactivated = u.IsActive && !next.IsActive
	*u = next
	return deactivated, nil
}

// SetPassword validates the password policy and stores the new hash.
func (u *User) SetPassword(password string, field string, hasher port.PasswordHasher) error {
	f := derr.FieldErrors{}
	validatePassword(f, field, password)
	if err := f.Err(); err != nil {
		return err
	}
	hash, err := hasher.Hash(password)
	if err != nil {
		return err
	}
	u.PasswordHash = hash
	return nil
}

func (u *User) PasswordMatches(password string, hasher port.PasswordHasher) bool {
	return hasher.Matches(u.PasswordHash, password)
}

func validateUsername(f derr.FieldErrors, username string) {
	if !usernamePattern.MatchString(username) {
		f.Add("username", "must be 3-100 characters of a-z, 0-9, '.', '_' or '-'")
	}
}

// validateEmail accepts nil (no email) or a single bare address.
func validateEmail(f derr.FieldErrors, email *string) {
	if email == nil {
		return
	}
	if len(*email) > maxEmailLen {
		f.Add("email", "must be at most 100 characters")
		return
	}
	// ParseAddress also accepts "Name <a@b.c>"; require the bare address.
	if addr, err := mail.ParseAddress(*email); err != nil || addr.Address != *email {
		f.Add("email", "must be a valid email address")
	}
}

func validateName(f derr.FieldErrors, name string) {
	switch n := utf8.RuneCountInString(name); {
	case n == 0:
		f.Add("name", "is required")
	case n > maxNameLen:
		f.Add("name", "must be at most 255 characters")
	}
}

func validateDateOfBirth(f derr.FieldErrors, dob, now time.Time) {
	switch {
	case dob.IsZero():
		f.Add("date_of_birth", "is required")
	case !dob.Before(shared.DateOf(now)):
		f.Add("date_of_birth", "must be in the past")
	case dob.Before(minDateOfBirth):
		f.Add("date_of_birth", "must be after 1900-01-01")
	}
}

func validatePassword(f derr.FieldErrors, field, password string) {
	switch n := len(password); {
	case n < minPasswordLen:
		f.Add(field, "must be at least 8 characters")
	case n > maxPasswordLen:
		f.Add(field, "must be at most 72 bytes")
	}
}

// NormalizeUsername trims and lower-cases a username.
func NormalizeUsername(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}

// normalizeEmail trims and lower-cases an email; empty means "no email".
func normalizeEmail(s *string) *string {
	if s == nil {
		return nil
	}
	e := strings.ToLower(strings.TrimSpace(*s))
	if e == "" {
		return nil
	}
	return &e
}
