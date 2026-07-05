package apierror

import (
	"errors"
	"goschool/internal/domain"
	"net/http"

	"github.com/go-chi/render"
	"github.com/rs/zerolog/log"
)

const (
	CodeInvalidBody  = "invalid_body"
	CodeInvalidParam = "invalid_param"
	CodeInvalidQuery = "invalid_query"
	CodeUnauthorized = "unauthorized"
	CodeForbidden    = "forbidden"
	CodeNotFound     = "not_found"
)

var statusMap = map[domain.ErrorType]int{
	domain.ErrTypeNotFound:     http.StatusNotFound,
	domain.ErrTypeInvalidInput: http.StatusBadRequest,
	domain.ErrTypeUnauthorized: http.StatusUnauthorized,
	domain.ErrTypeInternal:     http.StatusInternalServerError,
}

var (
	ErrUnknownInternal = &Error{
		Code:   "internal_error",
		Msg:    "An internal error has occurred",
		Status: http.StatusInternalServerError,
	}
	ErrUnauthorized = &Error{
		Code:   CodeUnauthorized,
		Msg:    "Unauthorized",
		Status: http.StatusUnauthorized,
	}
	ErrForbidden = &Error{
		Code:   CodeForbidden,
		Msg:    "Forbidden",
		Status: http.StatusForbidden,
	}
)

type Error struct {
	// Code is an machine-readable string that can be used to further categorize the error. e.g validation_error, user_notfound
	Code string `json:"code"`
	// Msg is a human-readable string that describes the error.
	Msg string `json:"message"`
	// Status is the HTTP status code that should be returned with the error.
	Status int `json:"status"`
}

func New(msg string, code string, status int) *Error {
	return &Error{
		Msg:    msg,
		Code:   code,
		Status: status,
	}
}
func (e *Error) Error() string {
	return e.Msg
}

func RenderError(w http.ResponseWriter, r *http.Request, err error) {
	var apiErr *Error

	// Fast path: err is already a fully-formed *APIError — render it as-is.
	if errors.As(err, &apiErr) {
		render.Status(r, apiErr.Status)
		render.JSON(w, r, apiErr)
		return
	}

	var domainErr *domain.Error
	// If the error is a domain.Error, map it to an APIError.
	if errors.As(err, &domainErr) {
		status, ok := statusMap[domainErr.Type]
		if !ok || domainErr.Type == domain.ErrTypeInternal {
			// Return a generic internal server error for unknown or internal domain errors.
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, ErrUnknownInternal)
			log.Error().Err(err).Msg("Unknown error")
			return
		}

		apiErr := &Error{
			Code:   domainErr.Code,
			Msg:    domainErr.Msg,
			Status: status,
		}
		render.Status(r, apiErr.Status)
		render.JSON(w, r, apiErr)
		log.Debug().Err(err).Msg("Domain error")
		return
	}

	// Fallback, return a generic internal server error.
	render.Status(r, http.StatusInternalServerError)
	render.JSON(w, r, ErrUnknownInternal)
	log.Error().Err(err).Msg("Unhandled internal error")
}
