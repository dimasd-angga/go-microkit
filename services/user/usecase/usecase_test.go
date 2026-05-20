package usecase

import (
	"context"
	"testing"
	"time"

	"github.com/dimasd-angga/go-microkit/internal/auth"
	apperr "github.com/dimasd-angga/go-microkit/internal/errors"
	"github.com/dimasd-angga/go-microkit/services/user/entity"
	"github.com/google/uuid"
)

type memRepo struct {
	byID    map[uuid.UUID]*entity.User
	byEmail map[string]*entity.User
}

func newMemRepo() *memRepo {
	return &memRepo{
		byID:    map[uuid.UUID]*entity.User{},
		byEmail: map[string]*entity.User{},
	}
}

func (m *memRepo) Create(_ context.Context, u *entity.User) error {
	if _, ok := m.byEmail[u.Email]; ok {
		return apperr.AlreadyExists("user", "email")
	}
	m.byID[u.ID] = u
	m.byEmail[u.Email] = u
	return nil
}

func (m *memRepo) FindByID(_ context.Context, id uuid.UUID) (*entity.User, error) {
	u, ok := m.byID[id]
	if !ok {
		return nil, apperr.NotFound("user", id)
	}
	return u, nil
}

func (m *memRepo) FindByEmail(_ context.Context, email string) (*entity.User, error) {
	u, ok := m.byEmail[email]
	if !ok {
		return nil, apperr.NotFound("user", email)
	}
	return u, nil
}

func TestRegister_HappyPath(t *testing.T) {
	signer := auth.NewSigner("test", time.Minute, time.Hour)
	uc := New(newMemRepo(), signer)

	u, err := uc.Register(context.Background(), RegisterInput{
		Email:    "Test@Example.COM",
		Name:     "Alice",
		Password: "supersecret",
	})
	if err != nil {
		t.Fatal(err)
	}
	if u.Email != "test@example.com" {
		t.Errorf("email not lowercased: %s", u.Email)
	}
	if u.PasswordHash == "supersecret" {
		t.Error("password not hashed")
	}
}

func TestRegister_RejectsBadEmail(t *testing.T) {
	uc := New(newMemRepo(), auth.NewSigner("test", time.Minute, time.Hour))
	_, err := uc.Register(context.Background(), RegisterInput{
		Email:    "nope",
		Name:     "Bob",
		Password: "supersecret",
	})
	e := apperr.As(err)
	if e == nil || e.Code != "invalid_email" {
		t.Fatalf("expected invalid_email, got %v", err)
	}
}

func TestLogin_WrongPassword_GenericError(t *testing.T) {
	signer := auth.NewSigner("test", time.Minute, time.Hour)
	uc := New(newMemRepo(), signer)

	_, _ = uc.Register(context.Background(), RegisterInput{
		Email: "a@b.com", Name: "Alice", Password: "rightpass1",
	})

	_, err := uc.Login(context.Background(), "a@b.com", "wrong")
	e := apperr.As(err)
	if e == nil || e.Code != "unauthorized" {
		t.Fatalf("expected unauthorized, got %v", err)
	}
}
