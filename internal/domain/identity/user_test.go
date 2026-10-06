package identity

import (
	"testing"
	"time"

	"goschool/internal/domain/derr"
	"goschool/internal/domain/shared"
)

var now = time.Date(2025, 10, 1, 8, 0, 0, 0, time.UTC)

func fieldsOf(t *testing.T, err error) map[string]string {
	t.Helper()
	e, ok := derr.As(err)
	if !ok || e.Kind != derr.KindInvalid {
		t.Fatalf("expected invalid error, got %v", err)
	}
	m := map[string]string{}
	for _, f := range e.Fields {
		m[f.Field] = f.Message
	}
	return m
}

func TestNewUserNormalizes(t *testing.T) {
	in := validInput("  John.Doe ")
	email := "  John@Example.COM "
	in.Email = &email
	in.Name = "  John Doe  "
	in.DateOfBirth = time.Date(2000, 5, 10, 23, 30, 0, 0, time.UTC)

	u, err := NewUser(in, shared.RoleTeacher, fakeHasher{}, now)
	if err != nil {
		t.Fatal(err)
	}
	if u.Username != "john.doe" || *u.Email != "john@example.com" || u.Name != "John Doe" {
		t.Fatalf("not normalized: %+v", u)
	}
	if !u.DateOfBirth.Equal(shared.NewDate(2000, 5, 10)) {
		t.Fatalf("DOB not truncated to date: %v", u.DateOfBirth)
	}
	if u.PasswordHash != "hashed:password123" || !u.IsActive {
		t.Fatalf("unexpected hash/active: %+v", u)
	}
}

func TestNewUserEmptyEmailBecomesNil(t *testing.T) {
	in := validInput("jane")
	empty := "   "
	in.Email = &empty
	u, err := NewUser(in, shared.RoleStudent, fakeHasher{}, now)
	if err != nil {
		t.Fatal(err)
	}
	if u.Email != nil {
		t.Fatalf("expected nil email, got %q", *u.Email)
	}
}

func TestNewUserValidation(t *testing.T) {
	badEmail := "not-an-email"
	in := CreateUserInput{
		Username:    "ab",
		Password:    "short",
		Email:       &badEmail,
		Name:        "  ",
		DateOfBirth: now.AddDate(0, 0, 1),
		Gender:      "robot",
	}
	_, err := NewUser(in, "superuser", fakeHasher{}, now)
	got := fieldsOf(t, err)
	for _, f := range []string{"username", "password", "email", "name", "date_of_birth", "gender", "role"} {
		if _, ok := got[f]; !ok {
			t.Errorf("expected field error for %q, got %v", f, got)
		}
	}
}

func TestApplyUpdate(t *testing.T) {
	u, _ := NewUser(validInput("jane"), shared.RoleStudent, fakeHasher{}, now)

	if _, err := u.ApplyUpdate(UpdateUserInput{}, now); !derr.Is(err, CodeNothingToUpdate) {
		t.Fatalf("expected nothing_to_update, got %v", err)
	}

	name := "Jane Roe"
	inactive := false
	deactivated, err := u.ApplyUpdate(UpdateUserInput{
		Name:     &name,
		Email:    shared.SetTo("JANE@x.io"),
		IsActive: &inactive,
	}, now)
	if err != nil {
		t.Fatal(err)
	}
	if !deactivated || u.Name != "Jane Roe" || *u.Email != "jane@x.io" || u.IsActive {
		t.Fatalf("update not applied: %+v deactivated=%v", u, deactivated)
	}

	if _, err := u.ApplyUpdate(UpdateUserInput{Email: shared.SetNull[string]()}, now); err != nil || u.Email != nil {
		t.Fatalf("email should be cleared: %v %v", err, u.Email)
	}

	// Only provided fields are validated: a stored value that would fail
	// today's rules does not block updating another field.
	u.Gender = "legacy"
	newName := "Jane Legacy"
	if _, err := u.ApplyUpdate(UpdateUserInput{Name: &newName}, now); err != nil {
		t.Fatalf("unrelated stored value must not be revalidated: %v", err)
	}
	u.Gender = GenderFemale

	// An invalid update leaves the user untouched.
	blank := ""
	before := *u
	if _, err := u.ApplyUpdate(UpdateUserInput{Name: &blank}, now); err == nil {
		t.Fatal("expected validation error")
	}
	if u.Name != before.Name {
		t.Fatal("invalid update must not mutate the user")
	}
}
