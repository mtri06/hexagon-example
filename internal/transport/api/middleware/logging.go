package middleware

import (
	"net/http"
	"time"

	chiMw "github.com/go-chi/chi/v5/middleware"
	"github.com/rs/zerolog/log"
)

func Logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		startTime := time.Now()

		ww := chiMw.NewWrapResponseWriter(w, r.ProtoMajor)
		next.ServeHTTP(ww, r)

		log.Info().
			Stringer("path", r.URL).
			Str("method", r.Method).
			Int("status", ww.Status()).
			Dur("duration", time.Since(startTime)).
			Msg("Request completed")
	})
}
