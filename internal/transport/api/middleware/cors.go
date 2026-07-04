package middleware

import (
	"goschool/internal/env"
	"net/http"

	"github.com/go-chi/cors"
)

func CORS() func(next http.Handler) http.Handler {
	opts := cors.Options{
		AllowedOrigins: env.Env.AllowedOrigins,
		AllowedMethods: []string{
			http.MethodHead,
			http.MethodGet,
			http.MethodPost,
			http.MethodPut,
			http.MethodPatch,
			http.MethodDelete,
		},
		AllowedHeaders:   []string{"*"},
		AllowCredentials: true,
		MaxAge:           7200,
	}
	return cors.New(opts).Handler
}
