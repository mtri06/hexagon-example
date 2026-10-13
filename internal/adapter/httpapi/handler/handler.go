// Package handler holds one HTTP handler per resource. Each handler depends on
// a narrow service interface declared next to it.
package handler

import (
	"net/http"

	"goschool/internal/adapter/httpapi/httpx"
	"goschool/internal/domain/shared"
)

// actorAndID reads the authenticated actor and the {id} path parameter.
func actorAndID(r *http.Request) (shared.Actor, int, error) {
	actor, err := httpx.Actor(r)
	if err != nil {
		return shared.Actor{}, 0, err
	}
	id, err := httpx.GetParamID(r, "id")
	return actor, id, err
}
