package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	apperr "github.com/dimasd-angga/go-microkit/internal/errors"
	"github.com/dimasd-angga/go-microkit/services/order/entity"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type OrderRepo interface {
	Create(ctx context.Context, o *entity.Order) error
	FindByID(ctx context.Context, id uuid.UUID) (*entity.Order, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, status entity.Status) (*entity.Order, error)
	ListByUser(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*entity.Order, int, error)
}

type Postgres struct{ pool *pgxpool.Pool }

func NewPostgres(pool *pgxpool.Pool) *Postgres { return &Postgres{pool: pool} }

func (r *Postgres) Create(ctx context.Context, o *entity.Order) error {
	const q = `
		INSERT INTO orders (id, user_id, status, total_cents, items, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`
	now := time.Now().UTC()
	o.CreatedAt, o.UpdatedAt = now, now

	items, err := json.Marshal(o.Items)
	if err != nil {
		return err
	}
	_, err = r.pool.Exec(ctx, q, o.ID, o.UserID, o.Status, o.TotalCents, items, o.CreatedAt, o.UpdatedAt)
	if err != nil {
		return fmt.Errorf("insert order: %w", err)
	}
	return nil
}

func (r *Postgres) FindByID(ctx context.Context, id uuid.UUID) (*entity.Order, error) {
	const q = `
		SELECT id, user_id, status, total_cents, items, created_at, updated_at
		FROM orders WHERE id = $1
	`
	return scanOrder(r.pool.QueryRow(ctx, q, id))
}

func (r *Postgres) UpdateStatus(ctx context.Context, id uuid.UUID, status entity.Status) (*entity.Order, error) {
	const q = `
		UPDATE orders SET status = $1, updated_at = now()
		WHERE id = $2
		RETURNING id, user_id, status, total_cents, items, created_at, updated_at
	`
	return scanOrder(r.pool.QueryRow(ctx, q, status, id))
}

func (r *Postgres) ListByUser(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*entity.Order, int, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}

	const q = `
		SELECT id, user_id, status, total_cents, items, created_at, updated_at,
		       count(*) OVER () AS total
		FROM orders
		WHERE user_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`
	rows, err := r.pool.Query(ctx, q, userID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var (
		orders []*entity.Order
		total  int
	)
	for rows.Next() {
		o := &entity.Order{}
		var raw []byte
		if err := rows.Scan(&o.ID, &o.UserID, &o.Status, &o.TotalCents, &raw, &o.CreatedAt, &o.UpdatedAt, &total); err != nil {
			return nil, 0, err
		}
		if err := json.Unmarshal(raw, &o.Items); err != nil {
			return nil, 0, err
		}
		orders = append(orders, o)
	}
	return orders, total, rows.Err()
}

func scanOrder(row pgx.Row) (*entity.Order, error) {
	o := &entity.Order{}
	var raw []byte
	err := row.Scan(&o.ID, &o.UserID, &o.Status, &o.TotalCents, &raw, &o.CreatedAt, &o.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperr.NotFound("order", "")
		}
		return nil, err
	}
	if err := json.Unmarshal(raw, &o.Items); err != nil {
		return nil, err
	}
	return o, nil
}
