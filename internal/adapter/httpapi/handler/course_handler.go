package handler

import (
	"context"
	"net/http"

	"github.com/go-chi/chi/v5"

	"goschool/internal/adapter/httpapi/dto"
	"goschool/internal/adapter/httpapi/httpx"
	"goschool/internal/adapter/httpapi/respond"
	"goschool/internal/domain/course"
	"goschool/internal/domain/shared"
)

type CourseService interface {
	List(ctx context.Context, actor shared.Actor, f course.Filter, p shared.PageRequest) (shared.Page[course.View], error)
	Get(ctx context.Context, actor shared.Actor, id int) (*course.View, error)
	Create(ctx context.Context, actor shared.Actor, in course.CreateInput) (*course.View, error)
	Update(ctx context.Context, actor shared.Actor, id int, in course.UpdateInput) (*course.View, error)
	Delete(ctx context.Context, actor shared.Actor, id int) error
}

type CourseHandler struct {
	svc CourseService
}

func NewCourseHandler(svc CourseService) *CourseHandler {
	return &CourseHandler{svc: svc}
}

// MountRoutes registers the course routes. Any authenticated user can read;
// CourseService restricts writes to admins.
func (h *CourseHandler) MountRoutes(_, protected chi.Router) {
	protected.Get("/courses", h.list)
	protected.Post("/courses", h.create)
	protected.Get("/courses/{id}", h.get)
	protected.Patch("/courses/{id}", h.update)
	protected.Delete("/courses/{id}", h.delete)
}

func (h *CourseHandler) list(w http.ResponseWriter, r *http.Request) {
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
	subjectID, err := httpx.GetQueryIntOptional(r, "subject_id")
	if err != nil {
		respond.Error(w, r, err)
		return
	}
	f := course.Filter{SubjectID: subjectID, Query: httpx.GetQueryOptional(r, "q")}
	if s := httpx.GetQueryOptional(r, "status"); s != nil {
		st := course.Status(*s)
		f.Status = &st
	}

	res, err := h.svc.List(r.Context(), actor, f, page)
	if err != nil {
		respond.Error(w, r, err)
		return
	}
	respond.JSON(w, http.StatusOK, dto.NewPageResponse(res, dto.NewCourse))
}

func (h *CourseHandler) get(w http.ResponseWriter, r *http.Request) {
	actor, id, err := actorAndID(r)
	if err != nil {
		respond.Error(w, r, err)
		return
	}
	v, err := h.svc.Get(r.Context(), actor, id)
	if err != nil {
		respond.Error(w, r, err)
		return
	}
	respond.JSON(w, http.StatusOK, dto.NewCourse(v))
}

func (h *CourseHandler) create(w http.ResponseWriter, r *http.Request) {
	actor, err := httpx.Actor(r)
	if err != nil {
		respond.Error(w, r, err)
		return
	}
	body, err := httpx.DecodeBody[dto.CourseCreateRequest](w, r)
	if err != nil {
		respond.Error(w, r, err)
		return
	}
	v, err := h.svc.Create(r.Context(), actor, body.ToDomain())
	if err != nil {
		respond.Error(w, r, err)
		return
	}
	respond.JSON(w, http.StatusCreated, dto.NewCourse(v))
}

func (h *CourseHandler) update(w http.ResponseWriter, r *http.Request) {
	actor, id, err := actorAndID(r)
	if err != nil {
		respond.Error(w, r, err)
		return
	}
	body, err := httpx.DecodeBody[dto.CourseUpdateRequest](w, r)
	if err != nil {
		respond.Error(w, r, err)
		return
	}
	v, err := h.svc.Update(r.Context(), actor, id, body.ToDomain())
	if err != nil {
		respond.Error(w, r, err)
		return
	}
	respond.JSON(w, http.StatusOK, dto.NewCourse(v))
}

func (h *CourseHandler) delete(w http.ResponseWriter, r *http.Request) {
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
