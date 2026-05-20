package usecase

import (
	"context"
	"strings"
	"time"

	"github.com/dimasd-angga/go-microkit/internal/auth"
	apperr "github.com/dimasd-angga/go-microkit/internal/errors"
	"github.com/dimasd-angga/go-microkit/internal/logger"
	"github.com/dimasd-angga/go-microkit/services/user/entity"
	"github.com/dimasd-angga/go-microkit/services/user/repository"
	"github.com/google/uuid"
)

type RegisterInput struct {
	Email    string
	Name     string
	Password string
}

type LoginResult struct {
	User   *entity.User
	Tokens *auth.TokenPair
}

type UserUsecase interface {
	Register(ctx context.Context, in RegisterInput) (*entity.User, error)
	Login(ctx context.Context, email, password string) (*LoginResult, error)
	GetByID(ctx context.Context, id uuid.UUID) (*entity.User, error)
}

type Usecase struct {
	repo   repository.UserRepo
	signer *auth.Signer
	clock  func() time.Time
}

func New(repo repository.UserRepo, signer *auth.Signer) *Usecase {
	return &Usecase{repo: repo, signer: signer, clock: time.Now}
}

func (u *Usecase) Register(ctx context.Context, in RegisterInput) (*entity.User, error) {
	in.Email = strings.ToLower(strings.TrimSpace(in.Email))
	in.Name = strings.TrimSpace(in.Name)

	if !validEmail(in.Email) {
		return nil, apperr.Invalid("invalid_email", "email is not valid")
	}
	if len(in.Name) < 2 {
		return nil, apperr.Invalid("invalid_name", "name must be at least 2 characters")
	}

	hash, err := auth.HashPassword(in.Password)
	if err != nil {
		return nil, apperr.Invalid("weak_password", err.Error())
	}

	user := &entity.User{
		ID:           uuid.New(),
		Email:        in.Email,
		Name:         in.Name,
		PasswordHash: hash,
		Role:         auth.RoleUser,
	}

	if err := u.repo.Create(ctx, user); err != nil {
		return nil, err
	}

	logger.Info(ctx, "user registered", "user_id", user.ID, "email", user.Email)
	return user, nil
}

func (u *Usecase) Login(ctx context.Context, email, password string) (*LoginResult, error) {
	email = strings.ToLower(strings.TrimSpace(email))

	user, err := u.repo.FindByEmail(ctx, email)
	if err != nil {
		// Don't reveal whether the email exists.
		if apperr.As(err) != nil {
			return nil, apperr.Unauthorized("invalid credentials")
		}
		return nil, err
	}

	if !auth.CheckPassword(user.PasswordHash, password) {
		return nil, apperr.Unauthorized("invalid credentials")
	}

	tokens, err := u.signer.Issue(user.ID, user.Email, user.Role)
	if err != nil {
		return nil, apperr.Internal("failed to issue tokens")
	}

	return &LoginResult{User: user, Tokens: tokens}, nil
}

func (u *Usecase) GetByID(ctx context.Context, id uuid.UUID) (*entity.User, error) {
	return u.repo.FindByID(ctx, id)
}

// validEmail is a tiny check; we don't validate deliverability here.
func validEmail(s string) bool {
	if len(s) < 3 || len(s) > 254 {
		return false
	}
	at := strings.IndexByte(s, '@')
	if at <= 0 || at == len(s)-1 {
		return false
	}
	if strings.IndexByte(s[at+1:], '.') < 0 {
		return false
	}
	return true
}
