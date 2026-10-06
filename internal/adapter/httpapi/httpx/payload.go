package httpx

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strings"

	"github.com/go-playground/validator/v10"

	"goschool/internal/adapter/httpapi/respond"
)

const maxBodyBytes = 1 << 20 // 1 MiB

var validate = validator.New(validator.WithRequiredStructEnabled())

func init() {
	validate.RegisterTagNameFunc(func(fld reflect.StructField) string {
		jsonTag := fld.Tag.Get("json")
		if jsonTag == "" || jsonTag == "-" {
			return fld.Name
		}
		return strings.SplitN(jsonTag, ",", 2)[0]
	})
}

// DecodeBody decodes a JSON body into T and checks its `validate` tags. Tags
// cover request shape only; business validation belongs to the domain.
func DecodeBody[T any](w http.ResponseWriter, r *http.Request) (T, error) {
	var payload T
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxBodyBytes))
	if err := dec.Decode(&payload); err != nil {
		return payload, respond.NewAPIError(http.StatusBadRequest, respond.CodeInvalidBody, decodeBodyErrMsg(err))
	}
	if err := validate.Struct(payload); err != nil {
		var vErrs validator.ValidationErrors
		if errors.As(err, &vErrs) {
			return payload, respond.NewAPIError(http.StatusBadRequest, respond.CodeInvalidBody,
				"Request body contains invalid data", fieldErrors(vErrs)...)
		}
		return payload, fmt.Errorf("validate body: %w", err)
	}
	return payload, nil
}

type validationMsgFn func(param string) string

var validationMessages = map[string]validationMsgFn{
	"required": func(string) string { return "is required" },
	"email":    func(string) string { return "must be a valid email address" },
	"min":      func(p string) string { return fmt.Sprintf("must be at least %s", p) },
	"max":      func(p string) string { return fmt.Sprintf("must be at most %s", p) },
	"len":      func(p string) string { return fmt.Sprintf("must be exactly %s characters long", p) },
	"gte":      func(p string) string { return fmt.Sprintf("must be greater than or equal to %s", p) },
	"lte":      func(p string) string { return fmt.Sprintf("must be less than or equal to %s", p) },
	"gt":       func(p string) string { return fmt.Sprintf("must be greater than %s", p) },
	"lt":       func(p string) string { return fmt.Sprintf("must be less than %s", p) },
	"oneof":    func(p string) string { return "must be one of: " + strings.ReplaceAll(p, " ", ", ") },
	"url":      func(string) string { return "must be a valid URL" },
	"uuid":     func(string) string { return "must be a valid UUID" },
	"numeric":  func(string) string { return "must be a numeric value" },
	"alphanum": func(string) string { return "must contain only alphanumeric characters" },
	"eqfield":  func(p string) string { return "must match " + p },
}

// fieldErrors converts validator errors to field errors named by their JSON
// path (e.g. `entries[2].score`), without the top-level struct name.
func fieldErrors(errs validator.ValidationErrors) []respond.FieldError {
	out := make([]respond.FieldError, 0, len(errs))
	for _, fe := range errs {
		field := fe.Namespace()
		if i := strings.IndexByte(field, '.'); i >= 0 {
			field = field[i+1:]
		}
		msg := fmt.Sprintf("is invalid (%s)", fe.Tag())
		if fn, ok := validationMessages[fe.Tag()]; ok {
			msg = fn(fe.Param())
		}
		out = append(out, respond.FieldError{Field: field, Message: msg})
	}
	return out
}

// decodeBodyErrMsg returns a user-friendly message for JSON decode errors.
func decodeBodyErrMsg(err error) string {
	var syntaxErr *json.SyntaxError
	var unmarshalErr *json.UnmarshalTypeError
	var maxBytesErr *http.MaxBytesError

	switch {
	case errors.Is(err, io.EOF):
		return "request body must not be empty"
	case errors.Is(err, io.ErrUnexpectedEOF):
		return "request body contains malformed JSON"
	case errors.As(err, &syntaxErr):
		return fmt.Sprintf("request body contains malformed JSON at position %d", syntaxErr.Offset)
	case errors.As(err, &unmarshalErr):
		return fmt.Sprintf("`%s` must be a %s", unmarshalErr.Field, unmarshalErr.Type.String())
	case errors.As(err, &maxBytesErr):
		return fmt.Sprintf("request body must not exceed %d bytes", maxBytesErr.Limit)
	default:
		var userErr interface{ UserMessage() string }
		if errors.As(err, &userErr) {
			return userErr.UserMessage()
		}
		return "request body is invalid"
	}
}
