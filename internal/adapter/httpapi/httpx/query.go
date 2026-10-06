package httpx

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"goschool/internal/adapter/httpapi/respond"
	"goschool/internal/domain/shared"
)

func invalidQuery(msg string) error {
	return respond.NewAPIError(http.StatusBadRequest, respond.CodeInvalidQuery, msg)
}

func GetQuery(r *http.Request, key string) (string, error) {
	value := r.URL.Query().Get(key)
	if value == "" {
		return "", invalidQuery(fmt.Sprintf("query `%v` is required", key))
	}
	return value, nil
}

func GetQueryOptional(r *http.Request, key string) *string {
	value := strings.TrimSpace(r.URL.Query().Get(key))
	if value == "" {
		return nil
	}
	return &value
}

func GetQueryInt(r *http.Request, key string) (int, error) {
	v, err := GetQueryIntOptional(r, key)
	if err != nil {
		return 0, err
	}
	if v == nil {
		return 0, invalidQuery(fmt.Sprintf("query `%v` is required", key))
	}
	return *v, nil
}

func GetQueryIntOrDefault(r *http.Request, key string, defaultVal int) (int, error) {
	v, err := GetQueryIntOptional(r, key)
	if err != nil || v == nil {
		return defaultVal, err
	}
	return *v, nil
}

func GetQueryIntOptional(r *http.Request, key string) (*int, error) {
	valueStr := r.URL.Query().Get(key)
	if valueStr == "" {
		return nil, nil
	}
	value, err := strconv.Atoi(valueStr)
	if err != nil {
		return nil, invalidQuery(fmt.Sprintf("query `%v` must be a valid integer", key))
	}
	return &value, nil
}

func GetQueryBoolOptional(r *http.Request, key string) (*bool, error) {
	valueStr := r.URL.Query().Get(key)
	if valueStr == "" {
		return nil, nil
	}
	value, err := strconv.ParseBool(valueStr)
	if err != nil {
		return nil, invalidQuery(fmt.Sprintf("query `%v` must be a valid boolean", key))
	}
	return &value, nil
}

func GetQueryList(r *http.Request, key string) []string {
	valStr := r.URL.Query().Get(key)
	if valStr == "" {
		return nil
	}
	return strings.Split(valStr, ",")
}

// GetPageRequest reads `page` and `page_size`. Out-of-range values are
// rejected rather than silently clamped.
func GetPageRequest(r *http.Request) (shared.PageRequest, error) {
	page, err := GetQueryIntOrDefault(r, "page", 1)
	if err != nil {
		return shared.PageRequest{}, err
	}
	size, err := GetQueryIntOrDefault(r, "page_size", shared.DefaultPageSize)
	if err != nil {
		return shared.PageRequest{}, err
	}
	if page < 1 {
		return shared.PageRequest{}, invalidQuery("query `page` must be at least 1")
	}
	if size < 1 || size > shared.MaxPageSize {
		return shared.PageRequest{}, invalidQuery(fmt.Sprintf("query `page_size` must be between 1 and %d", shared.MaxPageSize))
	}
	return shared.PageRequest{Page: page, PageSize: size}, nil
}
