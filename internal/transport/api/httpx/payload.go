package httpx

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strings"
	"time"

	"github.com/go-chi/render"
	"github.com/go-playground/validator/v10"
	"github.com/rs/zerolog/log"
)

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

func DecodeBody[T any](r *http.Request) (T, error) {
	var payload T
	err := render.DecodeJSON(r.Body, &payload)
	if err != nil {
		log.Error().Err(err).Msg("failed to decode request body")
		return payload, ErrInvalidBody.WithMsg(decodeBodyErrMsg(err))
	}
	err = validate.Struct(payload)
	if err != nil {
		if vErr, ok := err.(validator.ValidationErrors); ok {
			return payload, ErrValidationFailed.WithMsg(sanitizeValidationError(vErr))
		}
		log.Error().Err(err).Msg("validation error")
		return payload, ErrValidationFailed
	}
	return payload, nil
}

type validationMsgFn func(field, param string) string

var validationMessages = map[string]validationMsgFn{
	"required": func(field, _ string) string {
		return fmt.Sprintf("`%v` is required", field)
	},
	"email": func(field, _ string) string {
		return fmt.Sprintf("`%v` must be a valid email address", field)
	},
	"min": func(field, param string) string {
		return fmt.Sprintf("`%v` must be at least %s characters long", field, param)
	},
	"max": func(field, param string) string {
		return fmt.Sprintf("`%v` must be at most %s characters long", field, param)
	},
	"len": func(field, param string) string {
		return fmt.Sprintf("`%v` must be exactly %s characters long", field, param)
	},
	"gte": func(field, param string) string {
		return fmt.Sprintf("`%v` must be greater than or equal to %s", field, param)
	},
	"lte": func(field, param string) string {
		return fmt.Sprintf("`%v` must be less than or equal to %s", field, param)
	},
	"gt": func(field, param string) string {
		return fmt.Sprintf("`%v` must be greater than %s", field, param)
	},
	"lt": func(field, param string) string {
		return fmt.Sprintf("`%v` must be less than %s", field, param)
	},
	"oneof": func(field, param string) string {
		return fmt.Sprintf("`%v` must be one of: %s", field, strings.ReplaceAll(param, " ", ", "))
	},
	"url": func(field, _ string) string {
		return fmt.Sprintf("`%v` must be a valid URL", field)
	},
	"uuid": func(field, _ string) string {
		return fmt.Sprintf("`%v` must be a valid UUID", field)
	},
	"numeric": func(field, _ string) string {
		return fmt.Sprintf("`%v` must be a numeric value", field)
	},
	"alphanum": func(field, _ string) string {
		return fmt.Sprintf("`%v` must contain only alphanumeric characters", field)
	},
	"eqfield": func(field, param string) string {
		return fmt.Sprintf("`%v` must match %s", field, param)
	},
}

// decodeBodyErrMsg returns a user-friendly message for JSON decode errors.
func decodeBodyErrMsg(err error) string {
	var syntaxErr *json.SyntaxError
	var unmarshalErr *json.UnmarshalTypeError
	var timeParseErr *time.ParseError

	switch {
	case err == io.EOF || err == io.ErrUnexpectedEOF:
		return "request body must not be empty"
	case errors.As(err, &syntaxErr):
		return fmt.Sprintf("request body contains malformed JSON at position %d", syntaxErr.Offset)
	case errors.As(err, &unmarshalErr):
		return fmt.Sprintf("`%s` must be a %s", unmarshalErr.Field, unmarshalErr.Type.String())
	case errors.As(err, &timeParseErr):
		return fmt.Sprintf("invalid date/time format, expected RFC3339 (e.g. `%s`)", time.RFC3339)
	default:
		return "request body is invalid"
	}
}

// sanitizeValidationError returns a human-readable message for all validation errors.
func sanitizeValidationError(errs validator.ValidationErrors) string {
	var sb strings.Builder
	for _, fe := range errs {
		if fn, ok := validationMessages[fe.Tag()]; ok {
			sb.WriteString(fn(fe.Field(), fe.Param()))
		} else {
			fmt.Fprintf(&sb, "%s is invalid (%s)", fe.Field(), fe.Tag())
		}
		sb.WriteString("; ")
	}
	return strings.TrimSuffix(sb.String(), "; ")
}
