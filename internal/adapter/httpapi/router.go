// Package httpapi is the driving HTTP adapter: router, handlers, DTOs.
package httpapi

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"goschool/internal/adapter/httpapi/middleware"
	"goschool/internal/adapter/httpapi/respond"
)

const APIPrefix = "/api/v1"

// RouteMounter is implemented by every handler. public routes need no
// authentication; protected routes run behind the Auth middleware.
type RouteMounter interface {
	MountRoutes(public, protected chi.Router)
}

type RouterConfig struct {
	AllowedOrigins []string
	Verifier       middleware.TokenVerifier
	Handlers       []RouteMounter
}

func NewRouter(cfg RouterConfig) http.Handler {
	r := chi.NewRouter()
	r.Use(
		middleware.RequestID,
		middleware.Logging,
		middleware.Recoverer,
		middleware.CORS(cfg.AllowedOrigins),
	)
	r.NotFound(middleware.NotFound)
	r.MethodNotAllowed(middleware.MethodNotAllowed)

	r.Route(APIPrefix, func(public chi.Router) {
		public.NotFound(middleware.NotFound)
		public.MethodNotAllowed(middleware.MethodNotAllowed)
		public.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) {
			respond.JSON(w, http.StatusOK, map[string]string{"status": "ok"})
		})

		protected := public.With(middleware.Auth(cfg.Verifier))
		for _, h := range cfg.Handlers {
			h.MountRoutes(public, protected)
		}
	})
	return r
}
