package grpcx

import (
	apperr "github.com/dimasd-angga/go-microkit/internal/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ToStatus maps a domain error into a gRPC status. Anything we don't recognise
// becomes Internal so we don't leak details across the wire.
func ToStatus(err error) error {
	if err == nil {
		return nil
	}
	e := apperr.As(err)
	if e == nil {
		return status.Error(codes.Internal, "internal error")
	}
	var code codes.Code
	switch e.Kind {
	case apperr.KindNotFound:
		code = codes.NotFound
	case apperr.KindAlreadyExists, apperr.KindConflict:
		code = codes.AlreadyExists
	case apperr.KindInvalidInput:
		code = codes.InvalidArgument
	case apperr.KindUnauthorized:
		code = codes.Unauthenticated
	case apperr.KindForbidden:
		code = codes.PermissionDenied
	default:
		code = codes.Internal
	}
	return status.Error(code, e.Message)
}

func grpcInternal(msg string) error {
	return status.Error(codes.Internal, msg)
}
