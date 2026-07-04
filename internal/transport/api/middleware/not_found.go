package middleware

import (
	"goschool/internal/transport/api/httpx"
	"net/http"

	"github.com/go-chi/render"
)

func NotFound() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		render.Status(r, http.StatusNotFound)
		render.JSON(w, r, httpx.ErrNotFound.WithMsg("route not found"))
	}
}
