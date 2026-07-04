package internal

import (
	"fmt"
	"goschool/internal/db"
	"goschool/internal/domain"
	"goschool/internal/env"
	"goschool/internal/infras/postgres"
	"goschool/internal/logger"
	"goschool/internal/transport/api/handler"
	mw "goschool/internal/transport/api/middleware"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/rs/zerolog/log"
)

type Server struct{}

func NewServer() *Server {
	return &Server{}
}

func (s *Server) Start() error {
	env.Init()
	logger.Init()

	// Connect to Postgres
	dbClient := db.ConnectPostgres(db.DBConfig{
		Host:        env.Env.PgHost,
		Port:        env.Env.PgPort,
		User:        env.Env.PgUser,
		Password:    env.Env.PgPassword,
		Name:        env.Env.PgDBName,
		SSLMode:     env.Env.PgSSLMode,
		ConnTimeout: env.Env.PgConnTimeout,
	})
	log.Info().Msg("Connect to Postgres successfully")

	userRepo := postgres.NewUserRepository(dbClient)

	userSvc := domain.NewUserService(userRepo)

	userHandler := handler.NewStudentHandler(userSvc)

	r := chi.NewRouter()

	// Init middlewares
	r.Use(mw.CORS())
	r.Use(chimw.Timeout(20 * time.Second))
	r.Use(chimw.Recoverer)
	r.Use(mw.Logging)
	r.NotFound(mw.NotFound())

	userHandler.MountRoute(r)

	return http.ListenAndServe(fmt.Sprintf("%v:%v", env.Env.ListenIP, env.Env.ListenPort), r)
}
