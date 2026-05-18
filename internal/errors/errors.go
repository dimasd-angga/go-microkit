// Package errors defines typed domain errors that the gateway maps to HTTP
// status codes automatically. Services should return these from usecases
// instead of raw fmt.Errorf strings.
package errors

import (
	"errors"
	"fmt"
	"net/http"
)

type Kind int

const (
	KindUnknown Kind = iota
	KindNotFound
	KindAlreadyExists
	KindInvalidInput
	KindUnauthorized
	KindForbidden
	KindConflict
	KindInternal
)

type Error struct {
	Kind    Kind
	Code    string // stable machine-readable code, e.g. "user_not_found"
	Message string
	wrapped error
}

func (e *Error) Error() string {
	if e.wrapped != nil {
		return fmt.Sprintf("%s: %s: %v", e.Code, e.Message, e.wrapped)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

func (e *Error) Unwrap() error { return e.wrapped }

func (e *Error) Wrap(err error) *Error {
	e.wrapped = err
	return e
}

// HTTPStatus is consulted by the gateway error mapper.
func (e *Error) HTTPStatus() int {
	switch e.Kind {
	case KindNotFound:
		return http.StatusNotFound
	case KindAlreadyExists, KindConflict:
		return http.StatusConflict
	case KindInvalidInput:
		return http.StatusBadRequest
	case KindUnauthorized:
		return http.StatusUnauthorized
	case KindForbidden:
		return http.StatusForbidden
	default:
		return http.StatusInternalServerError
	}
}

// As extracts a *Error from any error in the chain. Returns nil if not found.
func As(err error) *Error {
	var e *Error
	if errors.As(err, &e) {
		return e
	}
	return nil
}

// Constructors — keep messages short. Detail goes in Wrap().

func NotFound(resource string, id any) *Error {
	return &Error{
		Kind:    KindNotFound,
		Code:    resource + "_not_found",
		Message: fmt.Sprintf("%s %v not found", resource, id),
	}
}

func AlreadyExists(resource, field string) *Error {
	return &Error{
		Kind:    KindAlreadyExists,
		Code:    resource + "_exists",
		Message: fmt.Sprintf("%s with this %s already exists", resource, field),
	}
}

func Invalid(code, msg string) *Error {
	return &Error{Kind: KindInvalidInput, Code: code, Message: msg}
}

func Unauthorized(msg string) *Error {
	if msg == "" {
		msg = "authentication required"
	}
	return &Error{Kind: KindUnauthorized, Code: "unauthorized", Message: msg}
}

func Forbidden(msg string) *Error {
	if msg == "" {
		msg = "forbidden"
	}
	return &Error{Kind: KindForbidden, Code: "forbidden", Message: msg}
}

func Internal(msg string) *Error {
	return &Error{Kind: KindInternal, Code: "internal_error", Message: msg}
}
