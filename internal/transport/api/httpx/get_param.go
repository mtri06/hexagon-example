package httpx

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
)

func GetParam(r *http.Request, key string) (string, error) {
	value := chi.URLParam(r, key)
	if value == "" {
		return "", ErrInvalidParam.WithMsg(fmt.Sprintf("parameter `%v` is required", key))
	}
	return value, nil
}

func GetParamInt(r *http.Request, key string) (int, error) {
	valueStr := chi.URLParam(r, key)
	if valueStr == "" {
		return 0, ErrInvalidParam.WithMsg(fmt.Sprintf("parameter `%v` is required", key))
	}
	value, err := strconv.Atoi(valueStr)
	if err != nil {
		return 0, ErrInvalidParam.WithMsg(fmt.Sprintf("parameter `%v` must be a valid integer", key))
	}
	return value, nil
}
