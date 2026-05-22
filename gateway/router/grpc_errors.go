package router

import (
	apperr "github.com/dimasd-angga/go-microkit/internal/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// fromGRPC maps a gRPC error back into a *apperr.Error so the HTTP responder
// can produce a clean JSON body.
func fromGRPC(err error) error {
	s, ok := status.FromError(err)
	if !ok {
		return apperr.Internal(err.Error())
	}
	switch s.Code() {
	case codes.NotFound:
		return &apperr.Error{Kind: apperr.KindNotFound, Code: "not_found", Message: s.Message()}
	case codes.AlreadyExists:
		return &apperr.Error{Kind: apperr.KindAlreadyExists, Code: "already_exists", Message: s.Message()}
	case codes.InvalidArgument:
		return apperr.Invalid("invalid_argument", s.Message())
	case codes.Unauthenticated:
		return apperr.Unauthorized(s.Message())
	case codes.PermissionDenied:
		return apperr.Forbidden(s.Message())
	default:
		return apperr.Internal(s.Message())
	}
}
