package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/dimasd-angga/go-microkit/internal/auth"
	apperr "github.com/dimasd-angga/go-microkit/internal/errors"
	"github.com/dimasd-angga/go-microkit/services/user/entity"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type UserRepo interface {
	Create(ctx context.Context, u *entity.User) error
	FindByID(ctx context.Context, id uuid.UUID) (*entity.User, error)
	FindByEmail(ctx context.Context, email string) (*entity.User, error)
}

type Postgres struct {
	pool *pgxpool.Pool
}

func NewPostgres(pool *pgxpool.Pool) *Postgres {
	return &Postgres{pool: pool}
}

func (r *Postgres) Create(ctx context.Context, u *entity.User) error {
	const q = `
		INSERT INTO users (id, email, name, password_hash, role, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`
	now := time.Now().UTC()
	u.CreatedAt = now
	u.UpdatedAt = now
	if u.Role == "" {
		u.Role = auth.RoleUser
	}

	_, err := r.pool.Exec(ctx, q, u.ID, u.Email, u.Name, u.PasswordHash, u.Role, u.CreatedAt, u.UpdatedAt)
	if err != nil {
		// 23505 = unique_violation
		if isUniqueViolation(err) {
			return apperr.AlreadyExists("user", "email")
		}
		return fmt.Errorf("insert user: %w", err)
	}
	return nil
}

func (r *Postgres) FindByID(ctx context.Context, id uuid.UUID) (*entity.User, error) {
	const q = `
		SELECT id, email, name, password_hash, role, created_at, updated_at
		FROM users WHERE id = $1
	`
	row := r.pool.QueryRow(ctx, q, id)
	return scan(row)
}

func (r *Postgres) FindByEmail(ctx context.Context, email string) (*entity.User, error) {
	const q = `
		SELECT id, email, name, password_hash, role, created_at, updated_at
		FROM users WHERE email = $1
	`
	row := r.pool.QueryRow(ctx, q, email)
	return scan(row)
}

func scan(row pgx.Row) (*entity.User, error) {
	var u entity.User
	err := row.Scan(&u.ID, &u.Email, &u.Name, &u.PasswordHash, &u.Role, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperr.NotFound("user", "")
		}
		return nil, err
	}
	return &u, nil
}

// pgx returns a wrapped error; this is a string check on purpose — the
// alternative is importing pgconn for one constant.
func isUniqueViolation(err error) bool {
	return err != nil && containsCode(err.Error(), "23505")
}

func containsCode(s, code string) bool {
	for i := 0; i+len(code) <= len(s); i++ {
		if s[i:i+len(code)] == code {
			return true
		}
	}
	return false
}
