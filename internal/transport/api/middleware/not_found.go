package middleware

import (
	"goschool/internal/transport/api/apierror"
	"net/http"

	"github.com/go-chi/render"
)

func NotFound() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		render.Status(r, http.StatusNotFound)
		render.JSON(w, r, apierror.New("Not Found", apierror.CodeNotFound, http.StatusNotFound))
	}
}
