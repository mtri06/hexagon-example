package httpx

import (
	"context"
	"net/http"

	"goschool/internal/adapter/httpapi/respond"
	"goschool/internal/domain/shared"
)

const (
	CookieAccessToken  = "access_token"
	CookieRefreshToken = "refresh_token"
)

type actorKey struct{}

func WithActor(ctx context.Context, a shared.Actor) context.Context {
	return context.WithValue(ctx, actorKey{}, a)
}

// Actor returns the authenticated caller. It fails with 401 when the route is
// not behind the Auth middleware.
func Actor(r *http.Request) (shared.Actor, error) {
	a, ok := r.Context().Value(actorKey{}).(shared.Actor)
	if !ok {
		return shared.Actor{}, respond.ErrUnauthorized
	}
	return a, nil
}
