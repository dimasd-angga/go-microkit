package handler

import (
	"context"

	"github.com/dimasd-angga/go-microkit/internal/grpcx"
	"github.com/dimasd-angga/go-microkit/services/user/entity"
	"github.com/dimasd-angga/go-microkit/services/user/usecase"
	userpb "github.com/dimasd-angga/go-microkit/proto/user"
	"github.com/google/uuid"
)

type GRPC struct {
	userpb.UnimplementedUserServiceServer
	uc usecase.UserUsecase
}

func NewGRPC(uc usecase.UserUsecase) *GRPC {
	return &GRPC{uc: uc}
}

func (h *GRPC) Register(ctx context.Context, req *userpb.RegisterRequest) (*userpb.UserResponse, error) {
	u, err := h.uc.Register(ctx, usecase.RegisterInput{
		Email:    req.GetEmail(),
		Name:     req.GetName(),
		Password: req.GetPassword(),
	})
	if err != nil {
		return nil, grpcx.ToStatus(err)
	}
	return toProto(u), nil
}

func (h *GRPC) GetByID(ctx context.Context, req *userpb.GetByIDRequest) (*userpb.UserResponse, error) {
	id, err := uuid.Parse(req.GetId())
	if err != nil {
		return nil, grpcx.ToStatus(err)
	}
	u, err := h.uc.GetByID(ctx, id)
	if err != nil {
		return nil, grpcx.ToStatus(err)
	}
	return toProto(u), nil
}

func (h *GRPC) Login(ctx context.Context, req *userpb.LoginRequest) (*userpb.LoginResponse, error) {
	res, err := h.uc.Login(ctx, req.GetEmail(), req.GetPassword())
	if err != nil {
		return nil, grpcx.ToStatus(err)
	}
	return &userpb.LoginResponse{
		AccessToken:      res.Tokens.Access,
		RefreshToken:     res.Tokens.Refresh,
		AccessExpiresAt:  res.Tokens.AccessExp.Format("2006-01-02T15:04:05Z07:00"),
		RefreshExpiresAt: res.Tokens.RefreshExp.Format("2006-01-02T15:04:05Z07:00"),
		User:             toProto(res.User),
	}, nil
}

func toProto(u *entity.User) *userpb.UserResponse {
	return &userpb.UserResponse{
		Id:        u.ID.String(),
		Email:     u.Email,
		Name:      u.Name,
		Role:      string(u.Role),
		CreatedAt: u.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}
