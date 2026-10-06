package middleware

import (
	"net/http"
	"runtime/debug"
	"time"

	chiMw "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"goschool/internal/adapter/httpapi/respond"
)

// RequestID assigns a request id (or keeps the incoming X-Request-Id), echoes
// it in the response and attaches a request-scoped logger to the context.
func RequestID(next http.Handler) http.Handler {
	return chiMw.RequestID(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := chiMw.GetReqID(r.Context())
		w.Header().Set(chiMw.RequestIDHeader, id)
		logger := log.With().Str("request_id", id).Logger()
		next.ServeHTTP(w, r.WithContext(logger.WithContext(r.Context())))
	}))
}

func Logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		ww := chiMw.NewWrapResponseWriter(w, r.ProtoMajor)
		next.ServeHTTP(ww, r)

		zerolog.Ctx(r.Context()).Info().
			Str("method", r.Method).
			Str("path", r.URL.Path).
			Int("status", ww.Status()).
			Dur("duration", time.Since(start)).
			Msg("request completed")
	})
}

// Recoverer turns a panic into a logged JSON 500.
func Recoverer(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				if rec == http.ErrAbortHandler {
					panic(rec)
				}
				zerolog.Ctx(r.Context()).Error().
					Interface("panic", rec).
					Bytes("stack", debug.Stack()).
					Msg("panic recovered")
				respond.JSON(w, respond.ErrInternal.Status, respond.ErrInternal)
			}
		}()
		next.ServeHTTP(w, r)
	})
}

func CORS(allowedOrigins []string) func(next http.Handler) http.Handler {
	return cors.New(cors.Options{
		AllowedOrigins: allowedOrigins,
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
	}).Handler
}

func NotFound(w http.ResponseWriter, r *http.Request) {
	respond.Error(w, r, respond.ErrNotFound)
}

func MethodNotAllowed(w http.ResponseWriter, r *http.Request) {
	respond.Error(w, r, respond.ErrMethod)
}
