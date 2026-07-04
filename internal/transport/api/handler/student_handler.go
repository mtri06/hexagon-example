package handler

import (
	"goschool/internal/domain"

	"github.com/go-chi/chi/v5"
)

type StudentHandler struct {
	svc domain.UserSvc
}

func NewStudentHandler(svc domain.UserSvc) *StudentHandler {
	return &StudentHandler{svc: svc}
}

func (h *StudentHandler) MountRoute(router chi.Router) {
}
