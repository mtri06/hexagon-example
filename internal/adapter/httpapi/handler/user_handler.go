package handler

import (
	"context"
	"net/http"

	"github.com/go-chi/chi/v5"

	"goschool/internal/adapter/httpapi/dto"
	"goschool/internal/adapter/httpapi/httpx"
	"goschool/internal/adapter/httpapi/middleware"
	"goschool/internal/adapter/httpapi/respond"
	"goschool/internal/domain/identity"
	"goschool/internal/domain/shared"
)

type UserService interface {
	List(ctx context.Context, actor shared.Actor, f identity.UserFilter, p shared.PageRequest) (shared.Page[identity.User], error)
	Get(ctx context.Context, actor shared.Actor, id int) (*identity.User, error)
	Update(ctx context.Context, actor shared.Actor, id int, in identity.UpdateUserInput) (*identity.User, error)
	ResetPassword(ctx context.Context, actor shared.Actor, id int, newPassword string) error
	CreateAdmin(ctx context.Context, actor shared.Actor, in identity.CreateUserInput) (*identity.User, error)
}

type UserHandler struct {
	svc UserService
}

func NewUserHandler(svc UserService) *UserHandler {
	return &UserHandler{svc: svc}
}

func (h *UserHandler) MountRoutes(_, protected chi.Router) {
	admin := protected.With(middleware.RequireRole(shared.RoleAdmin))
	admin.Get("/users", h.list)
	admin.Get("/users/{id}", h.get)
	admin.Patch("/users/{id}", h.update)
	admin.Post("/users/{id}/reset-password", h.resetPassword)
	admin.Post("/admins", h.createAdmin)
}

func (h *UserHandler) list(w http.ResponseWriter, r *http.Request) {
	actor, err := httpx.Actor(r)
	if err != nil {
		respond.Error(w, r, err)
		return
	}
	page, err := httpx.GetPageRequest(r)
	if err != nil {
		respond.Error(w, r, err)
		return
	}
	isActive, err := httpx.GetQueryBoolOptional(r, "is_active")
	if err != nil {
		respond.Error(w, r, err)
		return
	}
	f := identity.UserFilter{IsActive: isActive, Query: httpx.GetQueryOptional(r, "q")}
	if role := httpx.GetQueryOptional(r, "role"); role != nil {
		rl := shared.Role(*role)
		f.Role = &rl
	}

	res, err := h.svc.List(r.Context(), actor, f, page)
	if err != nil {
		respond.Error(w, r, err)
		return
	}
	respond.JSON(w, http.StatusOK, dto.NewPageResponse(res, dto.NewUser))
}

func (h *UserHandler) get(w http.ResponseWriter, r *http.Request) {
	actor, id, err := actorAndID(r)
	if err != nil {
		respond.Error(w, r, err)
		return
	}
	u, err := h.svc.Get(r.Context(), actor, id)
	if err != nil {
		respond.Error(w, r, err)
		return
	}
	respond.JSON(w, http.StatusOK, dto.NewUser(u))
}

func (h *UserHandler) update(w http.ResponseWriter, r *http.Request) {
	actor, id, err := actorAndID(r)
	if err != nil {
		respond.Error(w, r, err)
		return
	}
	body, err := httpx.DecodeBody[dto.UserUpdateRequest](w, r)
	if err != nil {
		respond.Error(w, r, err)
		return
	}
	u, err := h.svc.Update(r.Context(), actor, id, body.ToDomain())
	if err != nil {
		respond.Error(w, r, err)
		return
	}
	respond.JSON(w, http.StatusOK, dto.NewUser(u))
}

func (h *UserHandler) resetPassword(w http.ResponseWriter, r *http.Request) {
	actor, id, err := actorAndID(r)
	if err != nil {
		respond.Error(w, r, err)
		return
	}
	body, err := httpx.DecodeBody[dto.ResetPasswordRequest](w, r)
	if err != nil {
		respond.Error(w, r, err)
		return
	}
	if err := h.svc.ResetPassword(r.Context(), actor, id, body.NewPassword); err != nil {
		respond.Error(w, r, err)
		return
	}
	respond.NoContent(w)
}

func (h *UserHandler) createAdmin(w http.ResponseWriter, r *http.Request) {
	actor, err := httpx.Actor(r)
	if err != nil {
		respond.Error(w, r, err)
		return
	}
	body, err := httpx.DecodeBody[dto.UserCreateRequest](w, r)
	if err != nil {
		respond.Error(w, r, err)
		return
	}
	u, err := h.svc.CreateAdmin(r.Context(), actor, body.ToDomain())
	if err != nil {
		respond.Error(w, r, err)
		return
	}
	respond.JSON(w, http.StatusCreated, dto.NewUser(u))
}

// actorAndID reads the authenticated actor and the {id} path parameter.
func actorAndID(r *http.Request) (shared.Actor, int, error) {
	actor, err := httpx.Actor(r)
	if err != nil {
		return shared.Actor{}, 0, err
	}
	id, err := httpx.GetParamID(r, "id")
	return actor, id, err
}
