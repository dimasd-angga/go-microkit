package handler

import (
	"context"
	"fmt"

	"github.com/dimasd-angga/go-microkit/internal/logger"
	"github.com/dimasd-angga/go-microkit/internal/messaging"
	"github.com/dimasd-angga/go-microkit/services/notification/dedupe"
	"github.com/dimasd-angga/go-microkit/services/notification/sender"
)

type orderCreated struct {
	OrderID    string `json:"order_id"`
	UserID     string `json:"user_id"`
	TotalCents int64  `json:"total_cents"`
}

type orderPaid struct {
	OrderID    string `json:"order_id"`
	UserID     string `json:"user_id"`
	TotalCents int64  `json:"total_cents"`
}

type Handlers struct {
	Sender sender.Sender
	Dedupe *dedupe.Store
}

func (h *Handlers) OnOrderCreated(ctx context.Context, data []byte) error {
	var ev orderCreated
	if err := messaging.DecodeInto(data, &ev); err != nil {
		return err
	}
	claimed, err := h.Dedupe.Claim(ctx, "order_created:"+ev.OrderID)
	if err != nil {
		return err
	}
	if !claimed {
		logger.Debug(ctx, "skipping duplicate order.created", "order_id", ev.OrderID)
		return nil
	}

	// In a real system we'd look up the user's preferred channel. For the kit
	// we just send to a fake address derived from the user_id.
	return h.Sender.Send(ctx, sender.Message{
		To:      ev.UserID + "@example.test",
		Channel: sender.ChannelEmail,
		Subject: "Your order is being prepared",
		Body:    fmt.Sprintf("We received your order %s (total: %d cents).", ev.OrderID, ev.TotalCents),
	})
}

func (h *Handlers) OnOrderPaid(ctx context.Context, data []byte) error {
	var ev orderPaid
	if err := messaging.DecodeInto(data, &ev); err != nil {
		return err
	}
	claimed, err := h.Dedupe.Claim(ctx, "order_paid:"+ev.OrderID)
	if err != nil {
		return err
	}
	if !claimed {
		return nil
	}
	return h.Sender.Send(ctx, sender.Message{
		To:      ev.UserID + "@example.test",
		Channel: sender.ChannelEmail,
		Subject: "Payment received",
		Body:    fmt.Sprintf("Thanks! We've received payment for order %s.", ev.OrderID),
	})
}
