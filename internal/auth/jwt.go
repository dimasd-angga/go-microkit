package auth

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type Role string

const (
	RoleUser  Role = "user"
	RoleAdmin Role = "admin"
)

type Claims struct {
	UserID uuid.UUID `json:"uid"`
	Email  string    `json:"email"`
	Role   Role      `json:"role"`
	jwt.RegisteredClaims
}

type TokenPair struct {
	Access       string    `json:"access_token"`
	Refresh      string    `json:"refresh_token"`
	AccessExp    time.Time `json:"access_expires_at"`
	RefreshExp   time.Time `json:"refresh_expires_at"`
	TokenType    string    `json:"token_type"`
}

type Signer struct {
	secret     []byte
	accessTTL  time.Duration
	refreshTTL time.Duration
	issuer     string
}

func NewSigner(secret string, accessTTL, refreshTTL time.Duration) *Signer {
	return &Signer{
		secret:     []byte(secret),
		accessTTL:  accessTTL,
		refreshTTL: refreshTTL,
		issuer:     "go-microkit",
	}
}

func (s *Signer) Issue(userID uuid.UUID, email string, role Role) (*TokenPair, error) {
	now := time.Now()
	accessExp := now.Add(s.accessTTL)
	refreshExp := now.Add(s.refreshTTL)

	access, err := s.sign(Claims{
		UserID: userID,
		Email:  email,
		Role:   role,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    s.issuer,
			Subject:   userID.String(),
			ExpiresAt: jwt.NewNumericDate(accessExp),
			IssuedAt:  jwt.NewNumericDate(now),
			ID:        uuid.NewString(),
		},
	})
	if err != nil {
		return nil, err
	}

	// Refresh tokens carry only the subject + a separate jti. Keeps them small
	// and avoids leaking claim data if they're stored on the client.
	refresh, err := s.sign(Claims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    s.issuer + "/refresh",
			Subject:   userID.String(),
			ExpiresAt: jwt.NewNumericDate(refreshExp),
			IssuedAt:  jwt.NewNumericDate(now),
			ID:        uuid.NewString(),
		},
	})
	if err != nil {
		return nil, err
	}

	return &TokenPair{
		Access:     access,
		Refresh:    refresh,
		AccessExp:  accessExp,
		RefreshExp: refreshExp,
		TokenType:  "Bearer",
	}, nil
}

func (s *Signer) Verify(token string) (*Claims, error) {
	var c Claims
	parsed, err := jwt.ParseWithClaims(token, &c, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return s.secret, nil
	})
	if err != nil {
		return nil, err
	}
	if !parsed.Valid {
		return nil, errors.New("invalid token")
	}
	return &c, nil
}

func (s *Signer) sign(c Claims) (string, error) {
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, c)
	return tok.SignedString(s.secret)
}
