package httpx

import (
	"fmt"
	"goschool/internal/transport/api/apierror"
	"net/http"
	"strconv"
	"strings"
)

func GetQuery(r *http.Request, key string) (string, error) {
	value := r.URL.Query().Get(key)
	if value == "" {
		return "", apierror.New(fmt.Sprintf("query `%v` is required", key), apierror.CodeInvalidQuery, http.StatusBadRequest)
	}
	return value, nil
}

func GetQueryOrDefault(r *http.Request, key, defaultVal string) string {
	value := r.URL.Query().Get(key)
	if value == "" {
		return defaultVal
	}
	return value
}

func GetQueryInt(r *http.Request, key string) (int, error) {
	valueStr := r.URL.Query().Get(key)
	if valueStr == "" {
		return 0, apierror.New(fmt.Sprintf("query `%v` is required", key), apierror.CodeInvalidQuery, http.StatusBadRequest)
	}
	value, err := strconv.Atoi(valueStr)
	if err != nil {
		return 0, apierror.New(fmt.Sprintf("query `%v` must be a valid integer", key), apierror.CodeInvalidQuery, http.StatusBadRequest)
	}
	return value, nil
}

func GetQueryIntOrDefault(r *http.Request, key string, defaultVal int) (int, error) {
	valueStr := r.URL.Query().Get(key)
	if valueStr == "" {
		return defaultVal, nil
	}
	value, err := strconv.Atoi(valueStr)
	if err != nil {
		return 0, apierror.New(fmt.Sprintf("query `%v` must be a valid integer", key), apierror.CodeInvalidQuery, http.StatusBadRequest)
	}
	return value, nil
}

func GetQueryIntOptional(r *http.Request, key string) (*int, error) {
	valueStr := r.URL.Query().Get(key)
	if valueStr == "" {
		return nil, nil
	}
	value, err := strconv.Atoi(valueStr)
	if err != nil {
		return nil, apierror.New(fmt.Sprintf("query `%v` must be a valid integer", key), apierror.CodeInvalidQuery, http.StatusBadRequest)
	}
	return &value, nil
}

func GetQueryBool(r *http.Request, key string) (bool, error) {
	valueStr := r.URL.Query().Get(key)
	if valueStr == "" {
		return false, apierror.New(fmt.Sprintf("query `%v` is required", key), apierror.CodeInvalidQuery, http.StatusBadRequest)
	}
	value, err := strconv.ParseBool(valueStr)
	if err != nil {
		return false, apierror.New(fmt.Sprintf("query `%v` must be a valid boolean", key), apierror.CodeInvalidQuery, http.StatusBadRequest)
	}
	return value, nil
}

func GetQueryBoolOrDefault(r *http.Request, key string, defaultVal bool) (bool, error) {
	valueStr := r.URL.Query().Get(key)
	if valueStr == "" {
		return defaultVal, nil
	}
	value, err := strconv.ParseBool(valueStr)
	if err != nil {
		return false, apierror.New(fmt.Sprintf("query `%v` must be a valid boolean", key), apierror.CodeInvalidQuery, http.StatusBadRequest)
	}
	return value, nil
}

func GetQueryOptional(r *http.Request, key string) *string {
	value := r.URL.Query().Get(key)
	if value == "" {
		return nil
	}
	return &value
}

func GetQueryBoolOptional(r *http.Request, key string) (*bool, error) {
	valueStr := r.URL.Query().Get(key)
	if valueStr == "" {
		return nil, nil
	}
	value, err := strconv.ParseBool(valueStr)
	if err != nil {
		return nil, apierror.New(fmt.Sprintf("query `%v` must be a valid boolean", key), apierror.CodeInvalidQuery, http.StatusBadRequest)
	}
	return &value, nil
}

func GetQueryList(r *http.Request, key string) []string {
	valStr := r.URL.Query().Get(key)
	if valStr == "" {
		return nil
	}
	return strings.Split(r.URL.Query().Get(key), ",")
}
