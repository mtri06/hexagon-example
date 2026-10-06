package security

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"

	"goschool/internal/domain/shared"
)

type clock struct{ t time.Time }

func (c *clock) Now() time.Time { return c.t }

func TestBcrypt(t *testing.T) {
	h := BcryptHasher{Cost: bcrypt.MinCost}
	hash, err := h.Hash("password123")
	if err != nil {
		t.Fatal(err)
	}
	if !h.Matches(hash, "password123") || h.Matches(hash, "password124") {
		t.Fatal("bcrypt match mismatch")
	}
}

func TestJWTRoundTrip(t *testing.T) {
	c := &clock{time.Now().UTC()}
	j := NewJWT("0123456789abcdef0123456789abcdef", 15*time.Minute, c)
	want := shared.Actor{UserID: 42, Role: shared.RoleTeacher}

	tok, exp, err := j.IssueAccessToken(want)
	if err != nil {
		t.Fatal(err)
	}
	if !exp.Equal(c.t.Add(15 * time.Minute)) {
		t.Fatalf("unexpected expiry %v", exp)
	}
	got, err := j.VerifyAccessToken(tok)
	if err != nil || got != want {
		t.Fatalf("got %+v, %v", got, err)
	}

	c.t = c.t.Add(16 * time.Minute)
	if _, err := j.VerifyAccessToken(tok); err == nil {
		t.Fatal("expired token must be rejected")
	}
}

func TestJWTRejectsForeignTokens(t *testing.T) {
	c := &clock{time.Now().UTC()}
	j := NewJWT("0123456789abcdef0123456789abcdef", time.Minute, c)
	other := NewJWT("another-secret-another-secret-xx", time.Minute, c)

	tok, _, _ := other.IssueAccessToken(shared.Actor{UserID: 1, Role: shared.RoleAdmin})
	if _, err := j.VerifyAccessToken(tok); err == nil {
		t.Fatal("token signed with another secret must be rejected")
	}

	none, _ := jwt.NewWithClaims(jwt.SigningMethodNone, accessClaims{
		Role:             "admin",
		RegisteredClaims: jwt.RegisteredClaims{Subject: "1", ExpiresAt: jwt.NewNumericDate(c.t.Add(time.Minute))},
	}).SignedString(jwt.UnsafeAllowNoneSignatureType)
	if _, err := j.VerifyAccessToken(none); err == nil {
		t.Fatal("alg=none must be rejected")
	}
}

func TestRandomSecrets(t *testing.T) {
	a, _ := RandomSecrets{}.NewSecret()
	b, _ := RandomSecrets{}.NewSecret()
	if a == b || len(a) < 40 {
		t.Fatalf("weak secrets: %q %q", a, b)
	}
}
