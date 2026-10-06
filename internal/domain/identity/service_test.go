package identity

import (
	"context"
	"testing"
	"time"

	"goschool/internal/domain/derr"
	"goschool/internal/domain/shared"
)

var ctx = context.Background()

func TestLogin(t *testing.T) {
	e := newEnv()
	u := e.seedUser("alice", shared.RoleTeacher)

	if _, err := e.auth.Login(ctx, "nobody", "password123"); !derr.Is(err, CodeInvalidCredentials) {
		t.Fatalf("unknown user: got %v", err)
	}
	if _, err := e.auth.Login(ctx, "alice", "wrong-password"); !derr.Is(err, CodeInvalidCredentials) {
		t.Fatalf("wrong password: got %v", err)
	}

	res, err := e.auth.Login(ctx, " ALICE ", "password123")
	if err != nil {
		t.Fatal(err)
	}
	if res.User.ID != u.ID || res.Tokens.AccessToken == "" || res.Tokens.RefreshToken == "" {
		t.Fatalf("unexpected result: %+v", res)
	}
	stored, _ := e.tokens.GetByHashForUpdate(ctx, HashSecret(res.Tokens.RefreshToken), TokenTypeRefresh)
	if stored == nil || stored.Hash == res.Tokens.RefreshToken {
		t.Fatal("refresh token must be stored hashed")
	}

	inactive := false
	if _, err := e.svc.Update(ctx, shared.Actor{UserID: 999, Role: shared.RoleAdmin}, u.ID, UpdateUserInput{IsActive: &inactive}); err != nil {
		t.Fatal(err)
	}
	if _, err := e.auth.Login(ctx, "alice", "password123"); !derr.Is(err, CodeUserInactive) {
		t.Fatalf("inactive user: got %v", err)
	}
}

func TestRefreshRotationAndReuse(t *testing.T) {
	e := newEnv()
	u := e.seedUser("bob", shared.RoleStudent)
	res, _ := e.auth.Login(ctx, "bob", "password123")
	first := res.Tokens.RefreshToken

	second, err := e.auth.Refresh(ctx, first)
	if err != nil {
		t.Fatal(err)
	}
	if second.RefreshToken == first {
		t.Fatal("refresh must rotate the token")
	}
	if _, err := e.auth.Refresh(ctx, second.RefreshToken); err != nil {
		t.Fatalf("second rotation should work: %v", err)
	}

	// Replaying the first token is detected and revokes everything.
	if _, err := e.auth.Refresh(ctx, first); !derr.Is(err, CodeRefreshTokenReused) {
		t.Fatalf("expected reuse detection, got %v", err)
	}
	if n := e.tokens.activeFor(u.ID); n != 0 {
		t.Fatalf("expected all tokens revoked, %d still active", n)
	}
}

func TestRefreshFailures(t *testing.T) {
	e := newEnv()
	e.seedUser("carol", shared.RoleStudent)
	res, _ := e.auth.Login(ctx, "carol", "password123")

	if _, err := e.auth.Refresh(ctx, "garbage"); !derr.Is(err, CodeInvalidRefreshToken) {
		t.Fatalf("unknown token: got %v", err)
	}
	e.clock.now = e.clock.now.Add(25 * time.Hour)
	if _, err := e.auth.Refresh(ctx, res.Tokens.RefreshToken); !derr.Is(err, CodeRefreshTokenExpired) {
		t.Fatalf("expired token: got %v", err)
	}
}

func TestLogoutRevokesOnlyOwnToken(t *testing.T) {
	e := newEnv()
	a := e.seedUser("dave", shared.RoleStudent)
	b := e.seedUser("erin", shared.RoleStudent)
	resA, _ := e.auth.Login(ctx, "dave", "password123")

	// Another user cannot revoke dave's token.
	if err := e.auth.Logout(ctx, b.Actor(), resA.Tokens.RefreshToken); err != nil {
		t.Fatal(err)
	}
	if e.tokens.activeFor(a.ID) != 1 {
		t.Fatal("foreign logout must not revoke the token")
	}
	if err := e.auth.Logout(ctx, a.Actor(), resA.Tokens.RefreshToken); err != nil {
		t.Fatal(err)
	}
	if _, err := e.auth.Refresh(ctx, resA.Tokens.RefreshToken); !derr.Is(err, CodeInvalidRefreshToken) {
		t.Fatalf("revoked token: got %v", err)
	}
}

func TestChangePassword(t *testing.T) {
	e := newEnv()
	u := e.seedUser("frank", shared.RoleTeacher)
	_, _ = e.auth.Login(ctx, "frank", "password123")

	if err := e.auth.ChangePassword(ctx, u.Actor(), "wrong", "newpassword1"); !derr.Is(err, CodeCurrentPasswordIncorrect) {
		t.Fatalf("wrong current: got %v", err)
	}
	if err := e.auth.ChangePassword(ctx, u.Actor(), "password123", "password123"); !derr.IsKind(err, derr.KindInvalid) {
		t.Fatalf("same password: got %v", err)
	}
	if err := e.auth.ChangePassword(ctx, u.Actor(), "password123", "short"); !derr.IsKind(err, derr.KindInvalid) {
		t.Fatalf("weak password: got %v", err)
	}
	if err := e.auth.ChangePassword(ctx, u.Actor(), "password123", "newpassword1"); err != nil {
		t.Fatal(err)
	}
	if e.tokens.activeFor(u.ID) != 0 {
		t.Fatal("password change must revoke refresh tokens")
	}
	if _, err := e.auth.Login(ctx, "frank", "newpassword1"); err != nil {
		t.Fatalf("login with new password: %v", err)
	}
}

func TestUserServiceRequiresAdmin(t *testing.T) {
	e := newEnv()
	teacher := shared.Actor{UserID: 1, Role: shared.RoleTeacher}
	if _, err := e.svc.List(ctx, teacher, UserFilter{}, shared.PageRequest{}); !derr.IsKind(err, derr.KindForbidden) {
		t.Fatalf("got %v", err)
	}
	if _, err := e.svc.CreateAdmin(ctx, teacher, validInput("x123")); !derr.IsKind(err, derr.KindForbidden) {
		t.Fatalf("got %v", err)
	}
}

func TestUserServiceUpdateRules(t *testing.T) {
	e := newEnv()
	admin := e.seedUser("root", shared.RoleAdmin)
	other := e.seedUser("gina", shared.RoleStudent)
	email := "taken@x.io"
	_, _ = e.svc.Update(ctx, admin.Actor(), admin.ID, UpdateUserInput{Email: shared.SetTo(email)})

	inactive := false
	if _, err := e.svc.Update(ctx, admin.Actor(), admin.ID, UpdateUserInput{IsActive: &inactive}); !derr.Is(err, CodeCannotDeactivateSelf) {
		t.Fatalf("self-deactivation: got %v", err)
	}
	if _, err := e.svc.Update(ctx, admin.Actor(), other.ID, UpdateUserInput{Email: shared.SetTo("TAKEN@x.io")}); !derr.Is(err, CodeEmailAlreadyExists) {
		t.Fatalf("duplicate email: got %v", err)
	}
	// Re-saving one's own email is not a conflict.
	if _, err := e.svc.Update(ctx, admin.Actor(), admin.ID, UpdateUserInput{Email: shared.SetTo(email)}); err != nil {
		t.Fatalf("same email: %v", err)
	}
	if _, err := e.svc.Update(ctx, admin.Actor(), 404, UpdateUserInput{IsActive: &inactive}); !derr.Is(err, CodeUserNotFound) {
		t.Fatalf("missing user: got %v", err)
	}
}

func TestCreateAdminAndEnsureAdmin(t *testing.T) {
	e := newEnv()
	root := e.seedUser("root", shared.RoleAdmin)

	if _, err := e.svc.CreateAdmin(ctx, root.Actor(), validInput("root")); !derr.Is(err, CodeUsernameAlreadyExists) {
		t.Fatalf("duplicate username: got %v", err)
	}
	u, err := e.svc.CreateAdmin(ctx, root.Actor(), validInput("second"))
	if err != nil || u.Role != shared.RoleAdmin {
		t.Fatalf("create admin: %v %+v", err, u)
	}

	created, err := e.svc.EnsureAdmin(ctx, "bootstrap", "password123")
	if err != nil || !created {
		t.Fatalf("first ensure: %v %v", created, err)
	}
	created, err = e.svc.EnsureAdmin(ctx, "BOOTSTRAP", "password123")
	if err != nil || created {
		t.Fatalf("second ensure should be a no-op: %v %v", created, err)
	}
}
