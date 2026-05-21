package entity

import (
	"time"

	"github.com/google/uuid"
)

type Status string

const (
	StatusPending Status = "pending"
	StatusPaid    Status = "paid"
	StatusFailed  Status = "failed"
)

type Order struct {
	ID         uuid.UUID
	UserID     uuid.UUID
	Status     Status
	TotalCents int64
	Items      []Item
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

type Item struct {
	SKU       string
	Quantity  int32
	UnitCents int64
}

func (o *Order) Recalculate() {
	var sum int64
	for _, it := range o.Items {
		sum += it.UnitCents * int64(it.Quantity)
	}
	o.TotalCents = sum
}
