package entity

import (
	"time"

	"github.com/dimasd-angga/go-microkit/internal/auth"
	"github.com/google/uuid"
)

type User struct {
	ID           uuid.UUID
	Email        string
	Name         string
	PasswordHash string
	Role         auth.Role
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// Public strips the password hash before sending the user over the wire.
type Public struct {
	ID        uuid.UUID `json:"id"`
	Email     string    `json:"email"`
	Name      string    `json:"name"`
	Role      auth.Role `json:"role"`
	CreatedAt time.Time `json:"created_at"`
}

func (u *User) Public() Public {
	return Public{
		ID:        u.ID,
		Email:     u.Email,
		Name:      u.Name,
		Role:      u.Role,
		CreatedAt: u.CreatedAt,
	}
}
