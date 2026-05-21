package handler

import (
	"context"

	"github.com/dimasd-angga/go-microkit/internal/grpcx"
	orderpb "github.com/dimasd-angga/go-microkit/proto/order"
	"github.com/dimasd-angga/go-microkit/services/order/entity"
	"github.com/dimasd-angga/go-microkit/services/order/usecase"
	"github.com/google/uuid"
)

type GRPC struct {
	orderpb.UnimplementedOrderServiceServer
	uc *usecase.Usecase
}

func NewGRPC(uc *usecase.Usecase) *GRPC { return &GRPC{uc: uc} }

func (h *GRPC) Create(ctx context.Context, req *orderpb.CreateOrderRequest) (*orderpb.OrderResponse, error) {
	uid, err := uuid.Parse(req.GetUserId())
	if err != nil {
		return nil, grpcx.ToStatus(err)
	}
	items := make([]entity.Item, 0, len(req.GetItems()))
	for _, it := range req.GetItems() {
		items = append(items, entity.Item{
			SKU:       it.GetSku(),
			Quantity:  it.GetQuantity(),
			UnitCents: it.GetUnitCents(),
		})
	}
	o, err := h.uc.Create(ctx, uid, items)
	if err != nil {
		return nil, grpcx.ToStatus(err)
	}
	return toProto(o), nil
}

func (h *GRPC) GetByID(ctx context.Context, req *orderpb.GetByIDRequest) (*orderpb.OrderResponse, error) {
	id, err := uuid.Parse(req.GetId())
	if err != nil {
		return nil, grpcx.ToStatus(err)
	}
	o, err := h.uc.GetByID(ctx, id)
	if err != nil {
		return nil, grpcx.ToStatus(err)
	}
	return toProto(o), nil
}

func (h *GRPC) ListByUser(ctx context.Context, req *orderpb.ListByUserRequest) (*orderpb.ListOrdersResponse, error) {
	uid, err := uuid.Parse(req.GetUserId())
	if err != nil {
		return nil, grpcx.ToStatus(err)
	}
	orders, total, err := h.uc.ListByUser(ctx, uid, int(req.GetLimit()), int(req.GetOffset()))
	if err != nil {
		return nil, grpcx.ToStatus(err)
	}
	out := make([]*orderpb.OrderResponse, 0, len(orders))
	for _, o := range orders {
		out = append(out, toProto(o))
	}
	return &orderpb.ListOrdersResponse{Orders: out, Total: int32(total)}, nil
}

func (h *GRPC) MarkPaid(ctx context.Context, req *orderpb.MarkPaidRequest) (*orderpb.OrderResponse, error) {
	id, err := uuid.Parse(req.GetId())
	if err != nil {
		return nil, grpcx.ToStatus(err)
	}
	o, err := h.uc.MarkPaid(ctx, id)
	if err != nil {
		return nil, grpcx.ToStatus(err)
	}
	return toProto(o), nil
}

func toProto(o *entity.Order) *orderpb.OrderResponse {
	items := make([]*orderpb.OrderItem, 0, len(o.Items))
	for _, it := range o.Items {
		items = append(items, &orderpb.OrderItem{
			Sku:       it.SKU,
			Quantity:  it.Quantity,
			UnitCents: it.UnitCents,
		})
	}
	return &orderpb.OrderResponse{
		Id:         o.ID.String(),
		UserId:     o.UserID.String(),
		Status:     string(o.Status),
		TotalCents: o.TotalCents,
		Items:      items,
		CreatedAt:  o.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:  o.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}
