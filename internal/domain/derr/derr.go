// Package derr defines the single error type returned by the domain layer.
//
// Every business failure is a *Error with a Kind (which the transport layer maps
// to a status code), a stable machine-readable Code and a human-readable message.
// Anything that is not a *Error is treated as an unexpected internal failure.
package derr

import (
	"errors"
	"fmt"
	"sort"
)

type Kind uint8

const (
	// KindInvalid: input breaks an invariant.
	KindInvalid Kind = iota + 1
	// KindUnauthenticated: bad credentials or token.
	KindUnauthenticated
	// KindForbidden: the actor is not allowed to perform the operation.
	KindForbidden
	// KindNotFound: the entity, or an entity it references, does not exist.
	KindNotFound
	// KindConflict: uniqueness violation, or the entity is still referenced.
	KindConflict
	// KindRuleViolation: well-formed input that breaks a business rule.
	KindRuleViolation
)

func (k Kind) String() string {
	switch k {
	case KindInvalid:
		return "invalid"
	case KindUnauthenticated:
		return "unauthenticated"
	case KindForbidden:
		return "forbidden"
	case KindNotFound:
		return "not_found"
	case KindConflict:
		return "conflict"
	case KindRuleViolation:
		return "rule_violation"
	default:
		return "unknown"
	}
}

// Common error codes shared by all aggregates.
const (
	CodeValidationFailed = "validation_failed"
	CodeForbidden        = "forbidden"
	CodeUnauthenticated  = "unauthenticated"
)

type FieldError struct {
	Field   string
	Message string
}

type Error struct {
	Kind   Kind
	Code   string
	Msg    string
	Fields []FieldError
	Err    error
}

func (e *Error) Error() string {
	s := fmt.Sprintf("%s [%s]: %s", e.Kind, e.Code, e.Msg)
	if e.Err != nil {
		s += ": " + e.Err.Error()
	}
	return s
}

func (e *Error) Unwrap() error { return e.Err }

// Wrap returns a copy of e that wraps err as its cause.
func (e *Error) Wrap(err error) *Error {
	c := *e
	c.Err = err
	return &c
}

func newErr(kind Kind, code, msg string) *Error {
	return &Error{Kind: kind, Code: code, Msg: msg}
}

func Invalid(code, msg string, fields ...FieldError) *Error {
	e := newErr(KindInvalid, code, msg)
	e.Fields = fields
	return e
}

// InvalidField reports a single invalid field, using err's message (e.g. from a
// value type's Validate()) as the field message.
func InvalidField(field string, err error) *Error {
	return Invalid(CodeValidationFailed, "Invalid input", FieldError{Field: field, Message: err.Error()})
}

func Unauthenticated(code, msg string) *Error { return newErr(KindUnauthenticated, code, msg) }
func Forbidden(code, msg string) *Error       { return newErr(KindForbidden, code, msg) }
func NotFound(code, msg string) *Error        { return newErr(KindNotFound, code, msg) }
func Conflict(code, msg string) *Error        { return newErr(KindConflict, code, msg) }
func RuleViolation(code, msg string) *Error   { return newErr(KindRuleViolation, code, msg) }

// As returns the *Error in err's chain, if any.
func As(err error) (*Error, bool) {
	var e *Error
	if errors.As(err, &e) {
		return e, true
	}
	return nil, false
}

// Is reports whether err is a domain error with the given code.
func Is(err error, code string) bool {
	e, ok := As(err)
	return ok && e.Code == code
}

// IsKind reports whether err is a domain error of the given kind.
func IsKind(err error, kind Kind) bool {
	e, ok := As(err)
	return ok && e.Kind == kind
}

// FieldErrors collects per-field validation messages, keyed by field name.
type FieldErrors map[string]string

// Add records msg for field unless that field already has a message.
func (f FieldErrors) Add(field, msg string) {
	if _, exists := f[field]; !exists {
		f[field] = msg
	}
}

// AddErr records err's message for field; a nil err is ignored. Use it with
// value types' Validate() methods, whose messages omit the field name.
func (f FieldErrors) AddErr(field string, err error) {
	if err != nil {
		f.Add(field, err.Error())
	}
}

// Err returns nil when no field failed, or a KindInvalid error listing the
// fields in a stable (sorted) order.
func (f FieldErrors) Err() error {
	if len(f) == 0 {
		return nil
	}
	fields := make([]FieldError, 0, len(f))
	for field, msg := range f {
		fields = append(fields, FieldError{Field: field, Message: msg})
	}
	sort.Slice(fields, func(i, j int) bool { return fields[i].Field < fields[j].Field })
	return Invalid(CodeValidationFailed, "Invalid input", fields...)
}
