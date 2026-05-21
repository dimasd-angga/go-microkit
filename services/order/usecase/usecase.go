package usecase

import (
	"context"
	"time"

	apperr "github.com/dimasd-angga/go-microkit/internal/errors"
	"github.com/dimasd-angga/go-microkit/internal/logger"
	"github.com/dimasd-angga/go-microkit/internal/messaging"
	"github.com/dimasd-angga/go-microkit/services/order/entity"
	"github.com/dimasd-angga/go-microkit/services/order/repository"
	"github.com/google/uuid"
)

type OrderCreatedEvent struct {
	OrderID    uuid.UUID    `json:"order_id"`
	UserID     uuid.UUID    `json:"user_id"`
	TotalCents int64        `json:"total_cents"`
	Items      []entity.Item `json:"items"`
	OccurredAt time.Time    `json:"occurred_at"`
}

type OrderPaidEvent struct {
	OrderID    uuid.UUID `json:"order_id"`
	UserID     uuid.UUID `json:"user_id"`
	TotalCents int64     `json:"total_cents"`
	OccurredAt time.Time `json:"occurred_at"`
}

type Publisher interface {
	Publish(ctx context.Context, subject string, v any) error
}

type Usecase struct {
	repo repository.OrderRepo
	pub  Publisher
}

func New(repo repository.OrderRepo, pub Publisher) *Usecase {
	return &Usecase{repo: repo, pub: pub}
}

func (u *Usecase) Create(ctx context.Context, userID uuid.UUID, items []entity.Item) (*entity.Order, error) {
	if len(items) == 0 {
		return nil, apperr.Invalid("empty_order", "order must have at least one item")
	}
	for _, it := range items {
		if it.Quantity <= 0 || it.UnitCents < 0 || it.SKU == "" {
			return nil, apperr.Invalid("invalid_item", "every item needs a sku, positive quantity and non-negative unit price")
		}
	}

	o := &entity.Order{
		ID:     uuid.New(),
		UserID: userID,
		Status: entity.StatusPending,
		Items:  items,
	}
	o.Recalculate()

	if err := u.repo.Create(ctx, o); err != nil {
		return nil, err
	}

	// Fire-and-(almost)forget: log but don't fail the order if publish fails —
	// the order itself is persisted. A reconciliation job would replay these.
	if err := u.pub.Publish(ctx, messaging.SubjOrderCreated, OrderCreatedEvent{
		OrderID:    o.ID,
		UserID:     o.UserID,
		TotalCents: o.TotalCents,
		Items:      o.Items,
		OccurredAt: time.Now().UTC(),
	}); err != nil {
		logger.Warn(ctx, "publish order.created failed", "order_id", o.ID, "err", err)
	}

	return o, nil
}

func (u *Usecase) GetByID(ctx context.Context, id uuid.UUID) (*entity.Order, error) {
	return u.repo.FindByID(ctx, id)
}

func (u *Usecase) ListByUser(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*entity.Order, int, error) {
	return u.repo.ListByUser(ctx, userID, limit, offset)
}

func (u *Usecase) MarkPaid(ctx context.Context, id uuid.UUID) (*entity.Order, error) {
	o, err := u.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if o.Status == entity.StatusPaid {
		return o, nil // idempotent
	}
	if o.Status == entity.StatusFailed {
		return nil, apperr.Invalid("invalid_transition", "cannot mark a failed order as paid")
	}

	updated, err := u.repo.UpdateStatus(ctx, id, entity.StatusPaid)
	if err != nil {
		return nil, err
	}

	if err := u.pub.Publish(ctx, messaging.SubjOrderPaid, OrderPaidEvent{
		OrderID:    updated.ID,
		UserID:     updated.UserID,
		TotalCents: updated.TotalCents,
		OccurredAt: time.Now().UTC(),
	}); err != nil {
		logger.Warn(ctx, "publish order.paid failed", "order_id", updated.ID, "err", err)
	}
	return updated, nil
}
