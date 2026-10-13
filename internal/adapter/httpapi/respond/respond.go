// Package respond writes JSON responses. Error is the single place where
// errors are turned into HTTP responses.
package respond

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"os"

	"github.com/rs/zerolog"

	"goschool/internal/domain/derr"
)

// Transport-level error codes (domain codes come from the domain packages).
const (
	CodeInvalidBody      = "invalid_body"
	CodeInvalidParam     = "invalid_param"
	CodeInvalidQuery     = "invalid_query"
	CodeUnauthorized     = "unauthorized"
	CodeForbidden        = "forbidden"
	CodeNotFound         = "not_found"
	CodeMethodNotAllowed = "method_not_allowed"
	CodeInternal         = "internal_error"
	CodeTimeout          = "timeout"
)

// StatusClientClosedRequest is nginx's non-standard 499. It is only recorded
// in our access log: the client that would receive it is already gone.
const StatusClientClosedRequest = 499

type FieldError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// APIError is an error raised by the transport layer itself (bad JSON, bad
// query param, missing auth cookie, ...). It is rendered as-is.
type APIError struct {
	Status int          `json:"-"`
	Code   string       `json:"code"`
	Msg    string       `json:"message"`
	Fields []FieldError `json:"fields,omitempty"`
}

func (e *APIError) Error() string { return e.Msg }

func NewAPIError(status int, code, msg string, fields ...FieldError) *APIError {
	return &APIError{Status: status, Code: code, Msg: msg, Fields: fields}
}

var (
	ErrUnauthorized = NewAPIError(http.StatusUnauthorized, CodeUnauthorized, "Unauthorized")
	ErrForbidden    = NewAPIError(http.StatusForbidden, CodeForbidden, "Forbidden")
	ErrNotFound     = NewAPIError(http.StatusNotFound, CodeNotFound, "Not Found")
	ErrMethod       = NewAPIError(http.StatusMethodNotAllowed, CodeMethodNotAllowed, "Method Not Allowed")
	ErrInternal     = NewAPIError(http.StatusInternalServerError, CodeInternal, "An internal error has occurred")
	ErrTimeout      = NewAPIError(http.StatusServiceUnavailable, CodeTimeout, "The request took too long")
)

var kindStatus = map[derr.Kind]int{
	derr.KindInvalid:         http.StatusBadRequest,
	derr.KindUnauthenticated: http.StatusUnauthorized,
	derr.KindForbidden:       http.StatusForbidden,
	derr.KindNotFound:        http.StatusNotFound,
	derr.KindConflict:        http.StatusConflict,
	derr.KindRuleViolation:   http.StatusUnprocessableEntity,
}

// StatusOf returns the HTTP status for a domain error kind.
func StatusOf(k derr.Kind) int {
	if s, ok := kindStatus[k]; ok {
		return s
	}
	return http.StatusInternalServerError
}

// Error renders err. *APIError and *derr.Error are rendered with their code.
// A canceled request (client disconnected) is logged at debug level and
// answered with 499; a timeout becomes a 503. Anything else is logged and
// hidden behind a generic 500.
func Error(w http.ResponseWriter, r *http.Request, err error) {
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		JSON(w, apiErr.Status, apiErr)
		return
	}
	if de, ok := derr.As(err); ok {
		body := &APIError{Code: de.Code, Msg: de.Msg}
		for _, f := range de.Fields {
			body.Fields = append(body.Fields, FieldError{Field: f.Field, Message: f.Message})
		}
		JSON(w, StatusOf(de.Kind), body)
		return
	}
	logger := zerolog.Ctx(r.Context())
	switch {
	case isCanceled(r, err):
		logger.Debug().Err(err).Str("method", r.Method).Str("path", r.URL.Path).
			Msg("request canceled by client")
		w.WriteHeader(StatusClientClosedRequest)
		return
	case isTimeout(r, err):
		logger.Warn().Err(err).Str("method", r.Method).Str("path", r.URL.Path).
			Msg("request timed out")
		JSON(w, ErrTimeout.Status, ErrTimeout)
		return
	}
	logger.Error().Err(err).
		Str("method", r.Method).Str("path", r.URL.Path).
		Msg("unhandled error")
	JSON(w, ErrInternal.Status, ErrInternal)
}

// The request context is checked as well as err because drivers don't always
// keep the context error in the chain: bun's pgdriver turns a context deadline
// into a socket "i/o timeout" (*net.OpError), and doesn't abort a running
// query on plain cancellation.
func isCanceled(r *http.Request, err error) bool {
	return errors.Is(err, context.Canceled) || errors.Is(r.Context().Err(), context.Canceled)
}

// isTimeout also matches os.ErrDeadlineExceeded, which network deadlines
// (including pgdriver's read/write timeouts) report.
func isTimeout(r *http.Request, err error) bool {
	return errors.Is(err, context.DeadlineExceeded) ||
		errors.Is(err, os.ErrDeadlineExceeded) ||
		errors.Is(r.Context().Err(), context.DeadlineExceeded)
}

func JSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if v == nil {
		return
	}
	_ = json.NewEncoder(w).Encode(v)
}

func NoContent(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNoContent)
}
