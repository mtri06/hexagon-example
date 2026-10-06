// Package app is the composition root: it wires adapters, services and
// handlers together. cmd/goschool and the HTTP tests both build on it.
package app

import (
	"net/http"

	"github.com/uptrace/bun"

	"goschool/internal/adapter/httpapi"
	"goschool/internal/adapter/httpapi/handler"
	"goschool/internal/adapter/postgres"
	"goschool/internal/adapter/security"
	"goschool/internal/config"
	"goschool/internal/domain/identity"
	"goschool/internal/domain/port"
)

type App struct {
	Handler http.Handler
	Users   *identity.UserService
	Auth    *identity.AuthService
}

// Overrides lets tests swap slow or time-dependent dependencies.
type Overrides struct {
	Clock  port.Clock
	Hasher port.PasswordHasher
}

func New(cfg config.Config, db *bun.DB, o Overrides) *App {
	clock := o.Clock
	if clock == nil {
		clock = port.SystemClock{}
	}
	hasher := o.Hasher
	if hasher == nil {
		hasher = security.NewBcryptHasher()
	}
	jwt := security.NewJWT(cfg.JWT.Secret, cfg.JWT.AccessExpires, clock)
	tx := postgres.NewTxManager(db)

	userRepo := postgres.NewUserRepo(db)
	tokenRepo := postgres.NewTokenRepo(db)

	authSvc := identity.NewAuthService(identity.AuthServiceDeps{
		Users:      userRepo,
		Tokens:     tokenRepo,
		Tx:         tx,
		Hasher:     hasher,
		Issuer:     jwt,
		Secrets:    security.RandomSecrets{},
		Clock:      clock,
		RefreshTTL: cfg.JWT.RefreshExpires,
	})
	userSvc := identity.NewUserService(identity.UserServiceDeps{
		Users:  userRepo,
		Tokens: tokenRepo,
		Tx:     tx,
		Hasher: hasher,
		Clock:  clock,
	})

	router := httpapi.NewRouter(httpapi.RouterConfig{
		AllowedOrigins: cfg.HTTP.AllowedOrigins,
		Verifier:       jwt,
		Handlers: []httpapi.RouteMounter{
			handler.NewAuthHandler(authSvc, handler.CookieConfig{
				Secure:      cfg.IsProd(),
				RefreshPath: httpapi.APIPrefix + "/auth",
			}),
			handler.NewUserHandler(userSvc),
		},
	})

	return &App{Handler: router, Users: userSvc, Auth: authSvc}
}
