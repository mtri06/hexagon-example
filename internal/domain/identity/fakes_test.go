package identity

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"goschool/internal/domain/shared"
)

type fakeUsers struct {
	byID   map[int]*User
	nextID int
}

func newFakeUsers() *fakeUsers { return &fakeUsers{byID: map[int]*User{}, nextID: 1} }

func (f *fakeUsers) Create(_ context.Context, u *User) error {
	u.ID = f.nextID
	f.nextID++
	c := *u
	f.byID[u.ID] = &c
	return nil
}

func (f *fakeUsers) Update(_ context.Context, u *User) error {
	if _, ok := f.byID[u.ID]; !ok {
		return ErrUserNotFound
	}
	c := *u
	f.byID[u.ID] = &c
	return nil
}

func (f *fakeUsers) GetByID(_ context.Context, id int) (*User, error) {
	u, ok := f.byID[id]
	if !ok {
		return nil, ErrUserNotFound
	}
	c := *u
	return &c, nil
}

func (f *fakeUsers) GetByUsername(_ context.Context, username string) (*User, error) {
	for _, u := range f.byID {
		if u.Username == username {
			c := *u
			return &c, nil
		}
	}
	return nil, ErrUserNotFound
}

func (f *fakeUsers) UsernameExists(ctx context.Context, username string) (bool, error) {
	_, err := f.GetByUsername(ctx, username)
	return err == nil, nil
}

func (f *fakeUsers) EmailExists(_ context.Context, email string, exclude int) (bool, error) {
	for _, u := range f.byID {
		if u.ID != exclude && u.Email != nil && *u.Email == email {
			return true, nil
		}
	}
	return false, nil
}

func (f *fakeUsers) List(_ context.Context, flt UserFilter, p shared.PageRequest) ([]User, int, error) {
	var out []User
	for _, u := range f.byID {
		if flt.Role != nil && u.Role != *flt.Role {
			continue
		}
		if flt.Query != nil && !strings.Contains(u.Username, *flt.Query) {
			continue
		}
		out = append(out, *u)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	total := len(out)
	start := min(p.Offset(), total)
	end := min(start+p.PageSize, total)
	return out[start:end], total, nil
}

type fakeTokens struct {
	byID   map[int]*Token
	nextID int
}

func newFakeTokens() *fakeTokens { return &fakeTokens{byID: map[int]*Token{}, nextID: 1} }

func (f *fakeTokens) Create(_ context.Context, t *Token) error {
	t.ID = f.nextID
	f.nextID++
	c := *t
	f.byID[t.ID] = &c
	return nil
}

func (f *fakeTokens) GetByHashForUpdate(_ context.Context, hash string, typ TokenType) (*Token, error) {
	for _, t := range f.byID {
		if t.Hash == hash && t.Type == typ {
			c := *t
			return &c, nil
		}
	}
	return nil, ErrTokenNotFound
}

func (f *fakeTokens) Update(_ context.Context, t *Token) error {
	c := *t
	f.byID[t.ID] = &c
	return nil
}

func (f *fakeTokens) RevokeAllForUser(_ context.Context, userID int, typ TokenType) error {
	for _, t := range f.byID {
		if t.UserID == userID && t.Type == typ {
			t.IsRevoked = true
		}
	}
	return nil
}

func (f *fakeTokens) activeFor(userID int) int {
	n := 0
	for _, t := range f.byID {
		if t.UserID == userID && !t.IsRevoked && !t.IsUsed {
			n++
		}
	}
	return n
}

// fakeTx runs fn directly; rollback is not simulated.
type fakeTx struct{}

func (fakeTx) WithinTx(ctx context.Context, fn func(context.Context) error) error { return fn(ctx) }

type fakeHasher struct{}

func (fakeHasher) Hash(pw string) (string, error) { return "hashed:" + pw, nil }
func (fakeHasher) Matches(hash, pw string) bool   { return hash == "hashed:"+pw }

type fakeIssuer struct{ clock *fakeClock }

func (f fakeIssuer) IssueAccessToken(a shared.Actor) (string, time.Time, error) {
	return fmt.Sprintf("access:%d:%s", a.UserID, a.Role), f.clock.Now().Add(15 * time.Minute), nil
}

type fakeSecrets struct{ n int }

func (f *fakeSecrets) NewSecret() (string, error) {
	f.n++
	return fmt.Sprintf("secret-%d", f.n), nil
}

type fakeClock struct{ now time.Time }

func (c *fakeClock) Now() time.Time { return c.now }

type env struct {
	users  *fakeUsers
	tokens *fakeTokens
	clock  *fakeClock
	auth   *AuthService
	svc    *UserService
}

func newEnv() *env {
	e := &env{
		users:  newFakeUsers(),
		tokens: newFakeTokens(),
		clock:  &fakeClock{now: time.Date(2025, 10, 1, 8, 0, 0, 0, time.UTC)},
	}
	e.auth = NewAuthService(AuthServiceDeps{
		Users: e.users, Tokens: e.tokens, Tx: fakeTx{}, Hasher: fakeHasher{},
		Issuer: fakeIssuer{e.clock}, Secrets: &fakeSecrets{}, Clock: e.clock, RefreshTTL: 24 * time.Hour,
	})
	e.svc = NewUserService(UserServiceDeps{
		Users: e.users, Tokens: e.tokens, Tx: fakeTx{}, Hasher: fakeHasher{}, Clock: e.clock,
	})
	return e
}

func (e *env) seedUser(username string, role shared.Role) *User {
	u, err := NewUser(validInput(username), role, fakeHasher{}, e.clock.Now())
	if err != nil {
		panic(err)
	}
	_ = e.users.Create(context.Background(), u)
	return u
}

func validInput(username string) CreateUserInput {
	return CreateUserInput{
		Username:    username,
		Password:    "password123",
		Name:        "Test User",
		DateOfBirth: shared.NewDate(2000, time.May, 10),
		Gender:      GenderFemale,
	}
}
