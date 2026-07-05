package middleware

import (
	"context"
	"net/http"

	"goschool/internal/env"
	"goschool/internal/transport/api/apierror"
	"goschool/internal/transport/api/httpx"

	"github.com/golang-jwt/jwt/v5"
)

type contextKey string

const CtxKeyUserID contextKey = "user_id"
const CtxKeyUserRole contextKey = "user_role"

// Auth validates the JWT access token from the cookie and stores claims in context
func Auth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var tokenStr string

		cookie, err := r.Cookie(httpx.CookieAccessToken)
		if err != nil || cookie.Value == "" {
			apierror.RenderError(w, r, apierror.ErrUnauthorized)
			return
		}
		tokenStr = cookie.Value

		token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (any, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, apierror.ErrUnauthorized
			}
			return []byte(env.Env.JWTSecret), nil
		})
		if err != nil || !token.Valid {
			apierror.RenderError(w, r, apierror.ErrUnauthorized)
			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			apierror.RenderError(w, r, apierror.ErrUnauthorized)
			return
		}

		ctx := context.WithValue(r.Context(), CtxKeyUserID, claims["sub"])
		ctx = context.WithValue(ctx, CtxKeyUserRole, claims["role"])

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// RequireRole returns a middleware that allows only the specified roles
func RequireRole(roles ...string) func(http.Handler) http.Handler {
	allowed := make(map[string]struct{}, len(roles))
	for _, r := range roles {
		allowed[r] = struct{}{}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			role, ok := r.Context().Value(CtxKeyUserRole).(string)
			if !ok || role == "" {
				apierror.RenderError(w, r, apierror.ErrUnauthorized)
				return
			}

			if _, permitted := allowed[role]; !permitted {
				apierror.RenderError(w, r, apierror.ErrForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
