package domain

import "fmt"

type ErrorType string

const (
	ErrTypeNotFound     ErrorType = "not_found"
	ErrTypeInvalidInput ErrorType = "invalid_input"
	ErrTypeUnauthorized ErrorType = "unauthorized"
	ErrTypeInternal     ErrorType = "internal"
)

type Error struct {
	Msg  string
	Type ErrorType
	Code string
	Err  error
}

func (e *Error) Error() string {
	return fmt.Sprintf("Error: [%s:%s] %v: %v", e.Type, e.Code, e.Msg, e.Err)
}

func (e *Error) Unwrap() error {
	return e.Err
}
