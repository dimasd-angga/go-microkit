package usecase

import (
	"context"
	"errors"
	"testing"

	apperr "github.com/dimasd-angga/go-microkit/internal/errors"
	"github.com/dimasd-angga/go-microkit/services/order/entity"
	"github.com/google/uuid"
)

type memRepo struct {
	store map[uuid.UUID]*entity.Order
}

func newMemRepo() *memRepo { return &memRepo{store: map[uuid.UUID]*entity.Order{}} }

func (m *memRepo) Create(_ context.Context, o *entity.Order) error {
	m.store[o.ID] = o
	return nil
}
func (m *memRepo) FindByID(_ context.Context, id uuid.UUID) (*entity.Order, error) {
	o, ok := m.store[id]
	if !ok {
		return nil, apperr.NotFound("order", id)
	}
	return o, nil
}
func (m *memRepo) UpdateStatus(_ context.Context, id uuid.UUID, s entity.Status) (*entity.Order, error) {
	o, ok := m.store[id]
	if !ok {
		return nil, apperr.NotFound("order", id)
	}
	o.Status = s
	return o, nil
}
func (m *memRepo) ListByUser(_ context.Context, uid uuid.UUID, _, _ int) ([]*entity.Order, int, error) {
	var out []*entity.Order
	for _, o := range m.store {
		if o.UserID == uid {
			out = append(out, o)
		}
	}
	return out, len(out), nil
}

type stubPub struct {
	calls int
	fail  bool
}

func (p *stubPub) Publish(_ context.Context, _ string, _ any) error {
	p.calls++
	if p.fail {
		return errors.New("nats down")
	}
	return nil
}

func TestCreate_HappyPath(t *testing.T) {
	uc := New(newMemRepo(), &stubPub{})
	uid := uuid.New()

	o, err := uc.Create(context.Background(), uid, []entity.Item{
		{SKU: "AB-1", Quantity: 2, UnitCents: 1500},
		{SKU: "AB-2", Quantity: 1, UnitCents: 500},
	})
	if err != nil {
		t.Fatal(err)
	}
	if o.TotalCents != 3500 {
		t.Errorf("total = %d, want 3500", o.TotalCents)
	}
	if o.Status != entity.StatusPending {
		t.Errorf("status = %s", o.Status)
	}
}

func TestCreate_EmptyItemsRejected(t *testing.T) {
	uc := New(newMemRepo(), &stubPub{})
	_, err := uc.Create(context.Background(), uuid.New(), nil)
	if e := apperr.As(err); e == nil || e.Code != "empty_order" {
		t.Fatalf("want empty_order, got %v", err)
	}
}

func TestCreate_OrderPersistedEvenIfPublishFails(t *testing.T) {
	repo := newMemRepo()
	pub := &stubPub{fail: true}
	uc := New(repo, pub)

	o, err := uc.Create(context.Background(), uuid.New(), []entity.Item{
		{SKU: "x", Quantity: 1, UnitCents: 100},
	})
	if err != nil {
		t.Fatalf("expected create to succeed, got %v", err)
	}
	if _, ok := repo.store[o.ID]; !ok {
		t.Fatal("order not persisted")
	}
}

func TestMarkPaid_Idempotent(t *testing.T) {
	repo := newMemRepo()
	uc := New(repo, &stubPub{})
	o, _ := uc.Create(context.Background(), uuid.New(), []entity.Item{
		{SKU: "x", Quantity: 1, UnitCents: 100},
	})

	if _, err := uc.MarkPaid(context.Background(), o.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := uc.MarkPaid(context.Background(), o.ID); err != nil {
		t.Fatalf("second MarkPaid should be idempotent, got %v", err)
	}
}
