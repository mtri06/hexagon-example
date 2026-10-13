package app_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"

	"goschool/internal/adapter/postgres"
	"goschool/internal/adapter/postgres/pgtest"
	"goschool/internal/adapter/security"
	"goschool/internal/app"
	"goschool/internal/config"
	"goschool/internal/domain/derr"
	"goschool/internal/domain/identity"
	"goschool/internal/domain/shared"
)

const api = "/api/v1"

type client struct {
	t    *testing.T
	base string
	http *http.Client
}

func newClient(t *testing.T, base string) *client {
	jar, _ := cookiejar.New(nil)
	return &client{t: t, base: base, http: &http.Client{Jar: jar}}
}

type response struct {
	Status int
	Body   map[string]any
	Raw    *http.Response
}

func (c *client) do(method, path string, body any, extraCookies ...*http.Cookie) response {
	c.t.Helper()
	var rdr io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		rdr = bytes.NewReader(b)
	}
	req, _ := http.NewRequest(method, c.base+api+path, rdr)
	req.Header.Set("Content-Type", "application/json")
	for _, ck := range extraCookies {
		req.AddCookie(ck)
	}
	res, err := c.http.Do(req)
	if err != nil {
		c.t.Fatal(err)
	}
	defer res.Body.Close()
	out := response{Status: res.StatusCode, Raw: res}
	raw, _ := io.ReadAll(res.Body)
	if len(raw) > 0 {
		_ = json.Unmarshal(raw, &out.Body)
	}
	return out
}

func (c *client) expect(r response, status int, code string) {
	c.t.Helper()
	if r.Status != status {
		c.t.Fatalf("status = %d, want %d (body %v)", r.Status, status, r.Body)
	}
	if code != "" && r.Body["code"] != code {
		c.t.Fatalf("code = %v, want %q (body %v)", r.Body["code"], code, r.Body)
	}
}

func (c *client) login(username, password string) response {
	return c.do(http.MethodPost, "/auth/login", map[string]string{"username": username, "password": password})
}

func cookieFrom(res *http.Response, name string) *http.Cookie {
	for _, ck := range res.Cookies() {
		if ck.Name == name {
			return ck
		}
	}
	return nil
}

func setup(t *testing.T) (*httptest.Server, *app.App) {
	t.Helper()
	db := pgtest.DB(t)
	cfg := config.Config{
		Environment: config.EnvTest,
		JWT: config.JWT{
			Secret:         "test-secret-test-secret-test-secret",
			AccessExpires:  15 * time.Minute,
			RefreshExpires: time.Hour,
		},
	}
	a := app.New(cfg, db, app.Overrides{Hasher: security.BcryptHasher{Cost: bcrypt.MinCost}})
	srv := httptest.NewServer(a.Handler)
	t.Cleanup(srv.Close)

	if _, err := a.Users.EnsureAdmin(context.Background(), "root", "rootpassword"); err != nil {
		t.Fatal(err)
	}
	// A student account, inserted directly until the student aggregate exists.
	student, err := identity.NewUser(identity.CreateUserInput{
		Username: "stu", Password: "studentpass", Name: "Stu Dent",
		DateOfBirth: shared.NewDate(2010, 1, 2), Gender: identity.GenderMale,
	}, shared.RoleStudent, security.BcryptHasher{Cost: bcrypt.MinCost}, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if err := postgres.NewUserRepo(db).Create(context.Background(), student); err != nil {
		t.Fatal(err)
	}
	return srv, a
}

func TestRoutingBasics(t *testing.T) {
	srv, _ := setup(t)
	c := newClient(t, srv.URL)

	c.expect(c.do(http.MethodGet, "/healthz", nil), 200, "")
	c.expect(c.do(http.MethodGet, "/nope", nil), 404, "not_found")
	c.expect(c.do(http.MethodGet, "/auth/me", nil), 401, "unauthorized")
	c.expect(c.do(http.MethodGet, "/auth/me", nil, &http.Cookie{Name: "access_token", Value: "forged"}), 401, "unauthorized")
}

func TestAuthFlow(t *testing.T) {
	srv, _ := setup(t)
	c := newClient(t, srv.URL)

	c.expect(c.login("root", "wrong-password"), 401, identity.CodeInvalidCredentials)
	c.expect(c.do(http.MethodPost, "/auth/login", map[string]string{"username": "root"}), 400, "invalid_body")

	res := c.login("ROOT", "rootpassword")
	c.expect(res, 200, "")
	if res.Body["user"].(map[string]any)["role"] != "admin" {
		t.Fatalf("unexpected login body %v", res.Body)
	}
	access := cookieFrom(res.Raw, "access_token")
	firstRefresh := cookieFrom(res.Raw, "refresh_token")
	if access == nil || !access.HttpOnly || firstRefresh == nil || firstRefresh.Path != "/api/v1/auth" {
		t.Fatalf("bad cookies: %+v %+v", access, firstRefresh)
	}

	me := c.do(http.MethodGet, "/auth/me", nil)
	c.expect(me, 200, "")
	if me.Body["user"].(map[string]any)["username"] != "root" {
		t.Fatalf("me: %v", me.Body)
	}

	// Rotation: the jar picks up the new refresh cookie each time.
	c.expect(c.do(http.MethodPost, "/auth/refresh", nil), 204, "")
	c.expect(c.do(http.MethodPost, "/auth/refresh", nil), 204, "")

	// Replaying the first refresh token revokes the whole family.
	replay := newClient(t, srv.URL)
	c.expect(replay.do(http.MethodPost, "/auth/refresh", nil, firstRefresh), 401, identity.CodeRefreshTokenReused)
	c.expect(c.do(http.MethodPost, "/auth/refresh", nil), 401, identity.CodeInvalidRefreshToken)
}

func TestChangePasswordAndLogout(t *testing.T) {
	srv, _ := setup(t)
	c := newClient(t, srv.URL)
	c.expect(c.login("stu", "studentpass"), 200, "")

	r := c.do(http.MethodPut, "/auth/me/password", map[string]string{"current_password": "nope", "new_password": "newstudentpass"})
	c.expect(r, 400, identity.CodeCurrentPasswordIncorrect)

	r = c.do(http.MethodPut, "/auth/me/password", map[string]string{"current_password": "studentpass", "new_password": "short"})
	c.expect(r, 400, "validation_failed")
	if f := r.Body["fields"].([]any)[0].(map[string]any); f["field"] != "new_password" {
		t.Fatalf("expected new_password field error, got %v", r.Body)
	}

	c.expect(c.do(http.MethodPut, "/auth/me/password", map[string]string{"current_password": "studentpass", "new_password": "newstudentpass"}), 204, "")
	c.expect(c.login("stu", "studentpass"), 401, identity.CodeInvalidCredentials)
	c.expect(c.login("stu", "newstudentpass"), 200, "")

	c.expect(c.do(http.MethodPost, "/auth/logout", nil), 204, "")
	c.expect(c.do(http.MethodGet, "/auth/me", nil), 401, "")
	c.expect(c.do(http.MethodPost, "/auth/refresh", nil), 401, "")
}

func TestUserAdministration(t *testing.T) {
	srv, _ := setup(t)
	admin := newClient(t, srv.URL)
	admin.expect(admin.login("root", "rootpassword"), 200, "")

	student := newClient(t, srv.URL)
	student.expect(student.login("stu", "studentpass"), 200, "")
	student.expect(student.do(http.MethodGet, "/users", nil), 403, "forbidden")

	// Create admin: shape errors, domain validation errors, success, duplicate.
	admin.expect(admin.do(http.MethodPost, "/admins", map[string]any{"username": "x"}), 400, "invalid_body")
	admin.expect(admin.do(http.MethodPost, "/admins", map[string]any{
		"username": "ok_name", "password": "password123", "name": "N", "date_of_birth": "1990-13-45", "gender": "male",
	}), 400, "invalid_body")
	bad := admin.do(http.MethodPost, "/admins", map[string]any{
		"username": "x", "password": "password123", "name": "N", "date_of_birth": "2999-01-01", "gender": "male",
	})
	admin.expect(bad, 400, "validation_failed")
	if len(bad.Body["fields"].([]any)) != 2 {
		t.Fatalf("expected username + date_of_birth errors, got %v", bad.Body)
	}

	newAdmin := map[string]any{
		"username": "Second.Admin", "password": "password123", "email": "second@school.io",
		"name": "Second Admin", "date_of_birth": "1990-04-01", "gender": "female",
	}
	created := admin.do(http.MethodPost, "/admins", newAdmin)
	admin.expect(created, 201, "")
	if created.Body["username"] != "second.admin" || created.Body["date_of_birth"] != "1990-04-01" {
		t.Fatalf("unexpected created body %v", created.Body)
	}
	admin.expect(admin.do(http.MethodPost, "/admins", newAdmin), 409, identity.CodeUsernameAlreadyExists)
	id := int(created.Body["id"].(float64))

	// List + filters + pagination.
	list := admin.do(http.MethodGet, "/users?role=admin&page_size=1", nil)
	admin.expect(list, 200, "")
	meta := list.Body["meta"].(map[string]any)
	if meta["total"].(float64) != 2 || len(list.Body["data"].([]any)) != 1 {
		t.Fatalf("unexpected list %v", list.Body)
	}
	admin.expect(admin.do(http.MethodGet, "/users?page_size=1000", nil), 400, "invalid_query")
	admin.expect(admin.do(http.MethodGet, "/users?role=wizard", nil), 400, "validation_failed")

	// Get / patch.
	admin.expect(admin.do(http.MethodGet, fmt.Sprintf("/users/%d", id), nil), 200, "")
	admin.expect(admin.do(http.MethodGet, "/users/99999", nil), 404, identity.CodeUserNotFound)
	admin.expect(admin.do(http.MethodGet, "/users/abc", nil), 400, "invalid_param")
	admin.expect(admin.do(http.MethodPatch, fmt.Sprintf("/users/%d", id), map[string]any{}), 400, derr.CodeNothingToUpdate)

	patched := admin.do(http.MethodPatch, fmt.Sprintf("/users/%d", id), map[string]any{"email": nil, "name": "Renamed"})
	admin.expect(patched, 200, "")
	if patched.Body["email"] != nil || patched.Body["name"] != "Renamed" {
		t.Fatalf("patch not applied: %v", patched.Body)
	}

	me := admin.do(http.MethodGet, "/auth/me", nil)
	rootID := int(me.Body["user"].(map[string]any)["id"].(float64))
	admin.expect(admin.do(http.MethodPatch, fmt.Sprintf("/users/%d", rootID), map[string]any{"is_active": false}), 422, identity.CodeCannotDeactivateSelf)

	// Deactivating the student kills their session refresh and blocks login.
	stuList := admin.do(http.MethodGet, "/users?q=stu", nil)
	stuID := int(stuList.Body["data"].([]any)[0].(map[string]any)["id"].(float64))
	admin.expect(admin.do(http.MethodPatch, fmt.Sprintf("/users/%d", stuID), map[string]any{"is_active": false}), 200, "")
	student.expect(student.do(http.MethodPost, "/auth/refresh", nil), 401, "")
	student.expect(student.login("stu", "studentpass"), 401, identity.CodeUserInactive)

	// Reset password.
	admin.expect(admin.do(http.MethodPost, fmt.Sprintf("/users/%d/reset-password", id), map[string]any{"new_password": "brandnewpass"}), 204, "")
	second := newClient(t, srv.URL)
	second.expect(second.login("second.admin", "brandnewpass"), 200, "")
}
