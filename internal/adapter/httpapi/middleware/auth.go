package middleware

import (
	"net/http"

	"goschool/internal/adapter/httpapi/httpx"
	"goschool/internal/adapter/httpapi/respond"
	"goschool/internal/domain/shared"
)

// TokenVerifier validates an access token and returns its actor.
type TokenVerifier interface {
	VerifyAccessToken(token string) (shared.Actor, error)
}

// Auth reads the access token from the access_token cookie and stores the
// actor in the request context. Requests without a valid token get a 401.
func Auth(v TokenVerifier) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cookie, err := r.Cookie(httpx.CookieAccessToken)
			if err != nil || cookie.Value == "" {
				respond.Error(w, r, respond.ErrUnauthorized)
				return
			}
			actor, err := v.VerifyAccessToken(cookie.Value)
			if err != nil {
				respond.Error(w, r, respond.ErrUnauthorized)
				return
			}
			next.ServeHTTP(w, r.WithContext(httpx.WithActor(r.Context(), actor)))
		})
	}
}

// RequireRole lets only the given roles through. It is coarse gating; rules
// that depend on data are enforced by domain services.
func RequireRole(roles ...shared.Role) func(http.Handler) http.Handler {
	allowed := make(map[shared.Role]struct{}, len(roles))
	for _, r := range roles {
		allowed[r] = struct{}{}
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			actor, err := httpx.Actor(r)
			if err != nil {
				respond.Error(w, r, err)
				return
			}
			if _, ok := allowed[actor.Role]; !ok {
				respond.Error(w, r, respond.ErrForbidden)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
