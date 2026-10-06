// Package config loads application configuration from the environment
// (optionally seeded from a .env file). Each command validates only the
// sections it needs, so e.g. `migrate` does not require JWT settings.
package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/joho/godotenv"
)

const (
	EnvDev  = "dev"
	EnvTest = "test"
	EnvProd = "prod"
)

type Config struct {
	Environment string `validate:"required,oneof=dev test prod"`
	DB          DB
	HTTP        HTTP
	JWT         JWT
	Admin       Admin
}

type DB struct {
	Host        string        `validate:"required"`
	Port        int           `validate:"required,min=1,max=65535"`
	User        string        `validate:"required"`
	Password    string        `validate:"required"`
	Name        string        `validate:"required"`
	SSLMode     string        `validate:"required,oneof=disable allow prefer require verify-ca verify-full"`
	DialTimeout time.Duration `validate:"required"`
}

type HTTP struct {
	ListenIP       string   `validate:"required,ip"`
	ListenPort     int      `validate:"required,min=1,max=65535"`
	AllowedOrigins []string `validate:"dive,required"`
}

func (h HTTP) Addr() string { return fmt.Sprintf("%s:%d", h.ListenIP, h.ListenPort) }

type JWT struct {
	Secret         string        `validate:"required,min=32"`
	AccessExpires  time.Duration `validate:"required"`
	RefreshExpires time.Duration `validate:"required"`
}

type Admin struct {
	Username string `validate:"required"`
	Password string `validate:"required,min=8"`
}

var validate = validator.New(validator.WithRequiredStructEnabled())

// Load reads .env (if present) and the process environment. It only fails on
// malformed values; use the Validate* methods to check required sections.
func Load(envFiles ...string) (Config, error) {
	_ = godotenv.Load(envFiles...) // a missing .env is fine; the environment may be set directly

	r := &reader{}
	cfg := Config{
		Environment: r.str("ENVIRONMENT", EnvDev),
		DB: DB{
			Host:        r.str("PG_HOST", "localhost"),
			Port:        r.int("PG_PORT", 5432),
			User:        r.str("PG_USER", ""),
			Password:    r.str("PG_PASSWORD", ""),
			Name:        r.str("PG_DB_NAME", ""),
			SSLMode:     r.str("PG_SSL_MODE", "disable"),
			DialTimeout: r.duration("PG_CONN_TIMEOUT", 10*time.Second),
		},
		HTTP: HTTP{
			ListenIP:       r.str("LISTEN_IP", "0.0.0.0"),
			ListenPort:     r.int("LISTEN_PORT", 8080),
			AllowedOrigins: r.list("CORS_ALLOWED_ORIGINS"),
		},
		JWT: JWT{
			Secret:         r.str("JWT_SECRET", ""),
			AccessExpires:  r.duration("JWT_ACCESS_EXPIRES", 15*time.Minute),
			RefreshExpires: r.duration("JWT_REFRESH_EXPIRES", 7*24*time.Hour),
		},
		Admin: Admin{
			Username: r.str("ADMIN_USERNAME", ""),
			Password: r.str("ADMIN_PASSWORD", ""),
		},
	}
	if len(r.errs) > 0 {
		return Config{}, errors.Join(r.errs...)
	}
	if err := validate.Var(cfg.Environment, "required,oneof=dev test prod"); err != nil {
		return Config{}, fmt.Errorf("ENVIRONMENT: %w", err)
	}
	return cfg, nil
}

func (c Config) IsProd() bool { return c.Environment == EnvProd }

func (c Config) ValidateDB() error    { return check("database (PG_*)", c.DB) }
func (c Config) ValidateHTTP() error  { return check("http (LISTEN_*, CORS_*)", c.HTTP) }
func (c Config) ValidateJWT() error   { return check("jwt (JWT_*)", c.JWT) }
func (c Config) ValidateAdmin() error { return check("admin (ADMIN_*)", c.Admin) }

func check(section string, v any) error {
	if err := validate.Struct(v); err != nil {
		return fmt.Errorf("invalid %s config: %w", section, err)
	}
	return nil
}

type reader struct{ errs []error }

func (r *reader) str(key, def string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return def
}

func (r *reader) int(key string, def int) int {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		r.errs = append(r.errs, fmt.Errorf("%s: not an integer: %q", key, v))
	}
	return n
}

func (r *reader) duration(key string, def time.Duration) time.Duration {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		r.errs = append(r.errs, fmt.Errorf("%s: not a duration (e.g. 15m, 168h): %q", key, v))
	}
	return d
}

func (r *reader) list(key string) []string {
	var out []string
	for _, s := range strings.Split(os.Getenv(key), ",") {
		if s = strings.TrimSpace(s); s != "" {
			out = append(out, s)
		}
	}
	return out
}
