package httpx

import (
	"fmt"
	"goschool/internal/transport/api/apierror"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
)

func GetParam(r *http.Request, key string) (string, error) {
	value := chi.URLParam(r, key)
	if value == "" {
		return "", apierror.New(apierror.CodeInvalidParam, fmt.Sprintf("parameter `%v` is required", key), http.StatusBadRequest)
	}
	return value, nil
}

func GetParamInt(r *http.Request, key string) (int, error) {
	valueStr := chi.URLParam(r, key)
	if valueStr == "" {
		return 0, apierror.New(fmt.Sprintf("parameter `%v` is required", key), apierror.CodeInvalidParam, http.StatusBadRequest)
	}
	value, err := strconv.Atoi(valueStr)
	if err != nil {
		return 0, apierror.New(fmt.Sprintf("parameter `%v` must be a valid integer", key), apierror.CodeInvalidParam, http.StatusBadRequest)
	}
	return value, nil
}
