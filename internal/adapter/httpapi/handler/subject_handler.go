package handler

import (
	"context"
	"net/http"

	"github.com/go-chi/chi/v5"

	"goschool/internal/adapter/httpapi/dto"
	"goschool/internal/adapter/httpapi/httpx"
	"goschool/internal/adapter/httpapi/respond"
	"goschool/internal/domain/shared"
	"goschool/internal/domain/subject"
)

type SubjectService interface {
	List(ctx context.Context, actor shared.Actor, f subject.Filter, p shared.PageRequest) (shared.Page[subject.Subject], error)
	Get(ctx context.Context, actor shared.Actor, id int) (*subject.Subject, error)
	Create(ctx context.Context, actor shared.Actor, in subject.CreateInput) (*subject.Subject, error)
	Update(ctx context.Context, actor shared.Actor, id int, in subject.UpdateInput) (*subject.Subject, error)
	Delete(ctx context.Context, actor shared.Actor, id int) error
}

type SubjectHandler struct {
	svc SubjectService
}

func NewSubjectHandler(svc SubjectService) *SubjectHandler {
	return &SubjectHandler{svc: svc}
}

// MountRoutes registers the subject routes. Any authenticated user can read;
// SubjectService restricts writes to admins.
func (h *SubjectHandler) MountRoutes(_, protected chi.Router) {
	protected.Get("/subjects", h.list)
	protected.Post("/subjects", h.create)
	protected.Get("/subjects/{id}", h.get)
	protected.Patch("/subjects/{id}", h.update)
	protected.Delete("/subjects/{id}", h.delete)
}

func (h *SubjectHandler) list(w http.ResponseWriter, r *http.Request) {
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
	res, err := h.svc.List(r.Context(), actor, subject.Filter{Query: httpx.GetQueryOptional(r, "q")}, page)
	if err != nil {
		respond.Error(w, r, err)
		return
	}
	respond.JSON(w, http.StatusOK, dto.NewPageResponse(res, dto.NewSubject))
}

func (h *SubjectHandler) get(w http.ResponseWriter, r *http.Request) {
	actor, id, err := actorAndID(r)
	if err != nil {
		respond.Error(w, r, err)
		return
	}
	s, err := h.svc.Get(r.Context(), actor, id)
	if err != nil {
		respond.Error(w, r, err)
		return
	}
	respond.JSON(w, http.StatusOK, dto.NewSubject(s))
}

func (h *SubjectHandler) create(w http.ResponseWriter, r *http.Request) {
	actor, err := httpx.Actor(r)
	if err != nil {
		respond.Error(w, r, err)
		return
	}
	body, err := httpx.DecodeBody[dto.SubjectInput](w, r)
	if err != nil {
		respond.Error(w, r, err)
		return
	}
	s, err := h.svc.Create(r.Context(), actor, subject.CreateInput{Name: body.Name})
	if err != nil {
		respond.Error(w, r, err)
		return
	}
	respond.JSON(w, http.StatusCreated, dto.NewSubject(s))
}

func (h *SubjectHandler) update(w http.ResponseWriter, r *http.Request) {
	actor, id, err := actorAndID(r)
	if err != nil {
		respond.Error(w, r, err)
		return
	}
	body, err := httpx.DecodeBody[dto.SubjectInput](w, r)
	if err != nil {
		respond.Error(w, r, err)
		return
	}
	s, err := h.svc.Update(r.Context(), actor, id, subject.UpdateInput{Name: body.Name})
	if err != nil {
		respond.Error(w, r, err)
		return
	}
	respond.JSON(w, http.StatusOK, dto.NewSubject(s))
}

func (h *SubjectHandler) delete(w http.ResponseWriter, r *http.Request) {
	actor, id, err := actorAndID(r)
	if err != nil {
		respond.Error(w, r, err)
		return
	}
	if err := h.svc.Delete(r.Context(), actor, id); err != nil {
		respond.Error(w, r, err)
		return
	}
	respond.NoContent(w)
}
