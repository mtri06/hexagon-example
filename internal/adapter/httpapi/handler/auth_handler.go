package handler

import (
	"context"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"goschool/internal/adapter/httpapi/dto"
	"goschool/internal/adapter/httpapi/httpx"
	"goschool/internal/adapter/httpapi/respond"
	"goschool/internal/domain/derr"
	"goschool/internal/domain/identity"
	"goschool/internal/domain/shared"
)

type AuthService interface {
	Login(ctx context.Context, username, password string) (*identity.LoginResult, error)
	Refresh(ctx context.Context, rawRefreshToken string) (identity.TokenPair, error)
	Logout(ctx context.Context, actor shared.Actor, rawRefreshToken string) error
	Me(ctx context.Context, actor shared.Actor) (*identity.User, error)
	ChangePassword(ctx context.Context, actor shared.Actor, current, next string) error
}

// CookieConfig controls auth cookie attributes. RefreshPath must cover the
// refresh and logout endpoints.
type CookieConfig struct {
	Secure      bool
	RefreshPath string
}

type AuthHandler struct {
	svc     AuthService
	cookies CookieConfig
}

func NewAuthHandler(svc AuthService, cookies CookieConfig) *AuthHandler {
	return &AuthHandler{svc: svc, cookies: cookies}
}

func (h *AuthHandler) MountRoutes(public, protected chi.Router) {
	public.Post("/auth/login", h.login)
	public.Post("/auth/refresh", h.refresh)
	protected.Post("/auth/logout", h.logout)
	protected.Get("/auth/me", h.me)
	protected.Put("/auth/me/password", h.changePassword)
}

func (h *AuthHandler) login(w http.ResponseWriter, r *http.Request) {
	body, err := httpx.DecodeBody[dto.LoginRequest](w, r)
	if err != nil {
		respond.Error(w, r, err)
		return
	}
	res, err := h.svc.Login(r.Context(), body.Username, body.Password)
	if err != nil {
		respond.Error(w, r, err)
		return
	}
	h.setAuthCookies(w, res.Tokens)
	respond.JSON(w, http.StatusOK, dto.Me{User: dto.NewUser(res.User)})
}

func (h *AuthHandler) refresh(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie(httpx.CookieRefreshToken)
	if err != nil || cookie.Value == "" {
		respond.Error(w, r, identity.ErrInvalidRefreshToken)
		return
	}
	pair, err := h.svc.Refresh(r.Context(), cookie.Value)
	if err != nil {
		if derr.IsKind(err, derr.KindUnauthenticated) {
			h.clearAuthCookies(w)
		}
		respond.Error(w, r, err)
		return
	}
	h.setAuthCookies(w, pair)
	respond.NoContent(w)
}

func (h *AuthHandler) logout(w http.ResponseWriter, r *http.Request) {
	actor, err := httpx.Actor(r)
	if err != nil {
		respond.Error(w, r, err)
		return
	}
	var raw string
	if c, err := r.Cookie(httpx.CookieRefreshToken); err == nil {
		raw = c.Value
	}
	if err := h.svc.Logout(r.Context(), actor, raw); err != nil {
		respond.Error(w, r, err)
		return
	}
	h.clearAuthCookies(w)
	respond.NoContent(w)
}

func (h *AuthHandler) me(w http.ResponseWriter, r *http.Request) {
	actor, err := httpx.Actor(r)
	if err != nil {
		respond.Error(w, r, err)
		return
	}
	u, err := h.svc.Me(r.Context(), actor)
	if err != nil {
		respond.Error(w, r, err)
		return
	}
	respond.JSON(w, http.StatusOK, dto.Me{User: dto.NewUser(u)})
}

func (h *AuthHandler) changePassword(w http.ResponseWriter, r *http.Request) {
	actor, err := httpx.Actor(r)
	if err != nil {
		respond.Error(w, r, err)
		return
	}
	body, err := httpx.DecodeBody[dto.ChangePasswordRequest](w, r)
	if err != nil {
		respond.Error(w, r, err)
		return
	}
	if err := h.svc.ChangePassword(r.Context(), actor, body.CurrentPassword, body.NewPassword); err != nil {
		respond.Error(w, r, err)
		return
	}
	respond.NoContent(w)
}

func (h *AuthHandler) setAuthCookies(w http.ResponseWriter, p identity.TokenPair) {
	http.SetCookie(w, h.cookie(httpx.CookieAccessToken, p.AccessToken, "/", http.SameSiteLaxMode, p.AccessExpiresAt))
	http.SetCookie(w, h.cookie(httpx.CookieRefreshToken, p.RefreshToken, h.cookies.RefreshPath, http.SameSiteStrictMode, p.RefreshExpiresAt))
}

func (h *AuthHandler) clearAuthCookies(w http.ResponseWriter) {
	for _, c := range []*http.Cookie{
		h.cookie(httpx.CookieAccessToken, "", "/", http.SameSiteLaxMode, time.Time{}),
		h.cookie(httpx.CookieRefreshToken, "", h.cookies.RefreshPath, http.SameSiteStrictMode, time.Time{}),
	} {
		c.MaxAge = -1
		http.SetCookie(w, c)
	}
}

func (h *AuthHandler) cookie(name, value, path string, sameSite http.SameSite, expires time.Time) *http.Cookie {
	c := &http.Cookie{
		Name:     name,
		Value:    value,
		Path:     path,
		HttpOnly: true,
		Secure:   h.cookies.Secure,
		SameSite: sameSite,
	}
	if !expires.IsZero() {
		c.Expires = expires
		c.MaxAge = int(time.Until(expires).Seconds())
	}
	return c
}
