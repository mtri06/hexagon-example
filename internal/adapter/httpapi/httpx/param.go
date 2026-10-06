package httpx

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"goschool/internal/adapter/httpapi/respond"
)

func invalidParam(msg string) error {
	return respond.NewAPIError(http.StatusBadRequest, respond.CodeInvalidParam, msg)
}

func GetParam(r *http.Request, key string) (string, error) {
	value := chi.URLParam(r, key)
	if value == "" {
		return "", invalidParam(fmt.Sprintf("parameter `%v` is required", key))
	}
	return value, nil
}

// GetParamID parses a positive integer path parameter.
func GetParamID(r *http.Request, key string) (int, error) {
	valueStr, err := GetParam(r, key)
	if err != nil {
		return 0, err
	}
	value, err := strconv.Atoi(valueStr)
	if err != nil || value < 1 {
		return 0, invalidParam(fmt.Sprintf("parameter `%v` must be a positive integer", key))
	}
	return value, nil
}
