package postgres_test

import (
	"context"
	"testing"
	"time"

	"goschool/internal/adapter/postgres"
	"goschool/internal/adapter/postgres/pgtest"
	"goschool/internal/domain/derr"
	"goschool/internal/domain/identity"
	"goschool/internal/domain/shared"
)

var ctx = context.Background()

func newUser(username string, email *string, role shared.Role) *identity.User {
	return &identity.User{
		Username:     username,
		PasswordHash: "hash",
		Email:        email,
		Role:         role,
		Name:         "Name " + username,
		DateOfBirth:  shared.NewDate(2001, time.March, 4),
		Gender:       identity.GenderMale,
		IsActive:     true,
	}
}

func ptr[T any](v T) *T { return &v }

func TestUserRepoCRUD(t *testing.T) {
	db := pgtest.DB(t)
	repo := postgres.NewUserRepo(db)

	u := newUser("alice", ptr("alice@x.io"), shared.RoleTeacher)
	if err := repo.Create(ctx, u); err != nil {
		t.Fatal(err)
	}
	if u.ID == 0 || u.CreatedAt.IsZero() {
		t.Fatalf("id/timestamps not set: %+v", u)
	}

	got, err := repo.GetByUsername(ctx, "alice")
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != u.ID || *got.Email != "alice@x.io" || !got.DateOfBirth.Equal(u.DateOfBirth) || got.Role != shared.RoleTeacher {
		t.Fatalf("round trip mismatch: %+v", got)
	}

	got.Name = "Alice Updated"
	got.IsActive = false
	got.Email = nil
	before := got.UpdatedAt
	if err := repo.Update(ctx, got); err != nil {
		t.Fatal(err)
	}
	again, _ := repo.GetByID(ctx, u.ID)
	if again.Name != "Alice Updated" || again.IsActive || again.Email != nil {
		t.Fatalf("update not persisted: %+v", again)
	}
	if !again.UpdatedAt.After(before) {
		t.Fatalf("updated_at not bumped: %v -> %v", before, again.UpdatedAt)
	}

	if _, err := repo.GetByID(ctx, 9999); !derr.Is(err, identity.CodeUserNotFound) {
		t.Fatalf("missing user: got %v", err)
	}
	if err := repo.Update(ctx, &identity.User{ID: 9999, Role: shared.RoleAdmin, Gender: identity.GenderMale}); !derr.Is(err, identity.CodeUserNotFound) {
		t.Fatalf("update missing user: got %v", err)
	}
}

func TestUserRepoUniqueConstraints(t *testing.T) {
	db := pgtest.DB(t)
	repo := postgres.NewUserRepo(db)

	_ = repo.Create(ctx, newUser("bob", ptr("bob@x.io"), shared.RoleStudent))

	if err := repo.Create(ctx, newUser("bob", nil, shared.RoleStudent)); !derr.Is(err, identity.CodeUsernameAlreadyExists) {
		t.Fatalf("duplicate username: got %v", err)
	}
	if err := repo.Create(ctx, newUser("bob2", ptr("bob@x.io"), shared.RoleStudent)); !derr.Is(err, identity.CodeEmailAlreadyExists) {
		t.Fatalf("duplicate email: got %v", err)
	}
	// Several users without email are fine (partial unique index).
	if err := repo.Create(ctx, newUser("c1", nil, shared.RoleStudent)); err != nil {
		t.Fatal(err)
	}
	if err := repo.Create(ctx, newUser("c2", nil, shared.RoleStudent)); err != nil {
		t.Fatal(err)
	}

	if ok, _ := repo.UsernameExists(ctx, "bob"); !ok {
		t.Fatal("UsernameExists")
	}
	bob, _ := repo.GetByUsername(ctx, "bob")
	if ok, _ := repo.EmailExists(ctx, "bob@x.io", bob.ID); ok {
		t.Fatal("EmailExists must ignore the excluded user")
	}
	if ok, _ := repo.EmailExists(ctx, "bob@x.io", 0); !ok {
		t.Fatal("EmailExists")
	}
}

func TestUserRepoList(t *testing.T) {
	db := pgtest.DB(t)
	repo := postgres.NewUserRepo(db)
	for _, u := range []*identity.User{
		newUser("t_one", nil, shared.RoleTeacher),
		newUser("t_two", ptr("two@school.io"), shared.RoleTeacher),
		newUser("s_one", nil, shared.RoleStudent),
		newUser("50pct", nil, shared.RoleStudent),
	} {
		if err := repo.Create(ctx, u); err != nil {
			t.Fatal(err)
		}
	}

	teacher := shared.RoleTeacher
	users, total, err := repo.List(ctx, identity.UserFilter{Role: &teacher}, shared.PageRequest{Page: 1, PageSize: 1})
	if err != nil {
		t.Fatal(err)
	}
	if total != 2 || len(users) != 1 || users[0].Username != "t_one" {
		t.Fatalf("role filter/pagination: total=%d users=%+v", total, users)
	}

	_, total, _ = repo.List(ctx, identity.UserFilter{Query: ptr("SCHOOL")}, shared.PageRequest{Page: 1, PageSize: 10})
	if total != 1 {
		t.Fatalf("email search: total=%d", total)
	}
	// '_' and '%' are matched literally.
	_, total, _ = repo.List(ctx, identity.UserFilter{Query: ptr("%")}, shared.PageRequest{Page: 1, PageSize: 10})
	if total != 0 {
		t.Fatalf("'%%' must not be a wildcard: total=%d", total)
	}
}

func TestTokenRepo(t *testing.T) {
	db := pgtest.DB(t)
	users := postgres.NewUserRepo(db)
	tokens := postgres.NewTokenRepo(db)
	tx := postgres.NewTxManager(db)

	u := newUser("dora", nil, shared.RoleStudent)
	_ = users.Create(ctx, u)

	mk := func(hash string) *identity.Token {
		tok := &identity.Token{UserID: u.ID, Hash: hash, Type: identity.TokenTypeRefresh, ExpiresAt: time.Now().Add(time.Hour)}
		if err := tokens.Create(ctx, tok); err != nil {
			t.Fatal(err)
		}
		return tok
	}
	a, _ := mk("hash-a"), mk("hash-b")

	err := tx.WithinTx(ctx, func(ctx context.Context) error {
		got, err := tokens.GetByHashForUpdate(ctx, "hash-a", identity.TokenTypeRefresh)
		if err != nil {
			return err
		}
		if got.ID != a.ID {
			t.Fatalf("wrong token %d", got.ID)
		}
		got.IsUsed = true
		return tokens.Update(ctx, got)
	})
	if err != nil {
		t.Fatal(err)
	}
	got, _ := tokens.GetByHashForUpdate(ctx, "hash-a", identity.TokenTypeRefresh)
	if !got.IsUsed {
		t.Fatal("IsUsed not persisted")
	}

	if err := tokens.RevokeAllForUser(ctx, u.ID, identity.TokenTypeRefresh); err != nil {
		t.Fatal(err)
	}
	b, _ := tokens.GetByHashForUpdate(ctx, "hash-b", identity.TokenTypeRefresh)
	if !b.IsRevoked {
		t.Fatal("RevokeAllForUser did not revoke")
	}
	if _, err := tokens.GetByHashForUpdate(ctx, "nope", identity.TokenTypeRefresh); !derr.Is(err, identity.CodeTokenNotFound) {
		t.Fatalf("missing token: got %v", err)
	}
}

func TestTxRollback(t *testing.T) {
	db := pgtest.DB(t)
	users := postgres.NewUserRepo(db)
	tx := postgres.NewTxManager(db)

	sentinel := derr.RuleViolation("boom", "boom")
	err := tx.WithinTx(ctx, func(ctx context.Context) error {
		if err := users.Create(ctx, newUser("ghost", nil, shared.RoleStudent)); err != nil {
			return err
		}
		return sentinel
	})
	if err != sentinel {
		t.Fatalf("expected sentinel, got %v", err)
	}
	if ok, _ := users.UsernameExists(ctx, "ghost"); ok {
		t.Fatal("insert should have been rolled back")
	}
}
