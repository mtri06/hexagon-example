// Package security implements the hashing and token ports.
package security

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"

	"goschool/internal/domain/port"
	"goschool/internal/domain/shared"
)

// BcryptHasher implements port.PasswordHasher.
type BcryptHasher struct {
	Cost int
}

var _ port.PasswordHasher = BcryptHasher{}

func NewBcryptHasher() BcryptHasher { return BcryptHasher{Cost: bcrypt.DefaultCost} }

func (h BcryptHasher) Hash(password string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(password), h.Cost)
	if err != nil {
		return "", fmt.Errorf("hash password: %w", err)
	}
	return string(b), nil
}

func (h BcryptHasher) Matches(hash, password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}

// JWT issues and verifies HS256 access tokens with claims sub (user id) and role.
type JWT struct {
	secret []byte
	ttl    time.Duration
	clock  port.Clock
}

var _ port.TokenIssuer = (*JWT)(nil)

func NewJWT(secret string, ttl time.Duration, clock port.Clock) *JWT {
	return &JWT{secret: []byte(secret), ttl: ttl, clock: clock}
}

type accessClaims struct {
	Role string `json:"role"`
	jwt.RegisteredClaims
}

func (j *JWT) IssueAccessToken(actor shared.Actor) (string, time.Time, error) {
	now := j.clock.Now()
	exp := now.Add(j.ttl)
	claims := accessClaims{
		Role: string(actor.Role),
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   strconv.Itoa(actor.UserID),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(exp),
		},
	}
	signed, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(j.secret)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("sign access token: %w", err)
	}
	return signed, exp, nil
}

var errInvalidToken = errors.New("invalid access token")

func (j *JWT) VerifyAccessToken(token string) (shared.Actor, error) {
	var claims accessClaims
	_, err := jwt.ParseWithClaims(token, &claims,
		func(*jwt.Token) (any, error) { return j.secret, nil },
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithExpirationRequired(),
		jwt.WithTimeFunc(j.clock.Now),
	)
	if err != nil {
		return shared.Actor{}, fmt.Errorf("%w: %v", errInvalidToken, err)
	}
	id, err := strconv.Atoi(claims.Subject)
	role := shared.Role(claims.Role)
	if err != nil || id < 1 || role.Validate() != nil {
		return shared.Actor{}, errInvalidToken
	}
	return shared.Actor{UserID: id, Role: role}, nil
}

// RandomSecrets implements port.SecretGenerator with 32 random bytes.
type RandomSecrets struct{}

var _ port.SecretGenerator = RandomSecrets{}

func (RandomSecrets) NewSecret() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate secret: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}
