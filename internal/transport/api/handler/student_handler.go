package handler

import (
	"github.com/go-chi/chi/v5"
)

type UserSvc interface{}

type StudentHandler struct {
	svc UserSvc
}

func NewStudentHandler(svc UserSvc) *StudentHandler {
	return &StudentHandler{svc: svc}
}

func (h *StudentHandler) MountRoute(router chi.Router) {
}
