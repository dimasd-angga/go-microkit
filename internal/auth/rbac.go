package auth

import (
	"context"

	"github.com/google/uuid"
)

type ctxKey struct{}

type Principal struct {
	UserID uuid.UUID
	Email  string
	Role   Role
}

func WithPrincipal(ctx context.Context, p Principal) context.Context {
	return context.WithValue(ctx, ctxKey{}, p)
}

func PrincipalFrom(ctx context.Context) (Principal, bool) {
	p, ok := ctx.Value(ctxKey{}).(Principal)
	return p, ok
}

// RequireRole returns true if the principal has at least one of the given roles.
func RequireRole(ctx context.Context, roles ...Role) bool {
	p, ok := PrincipalFrom(ctx)
	if !ok {
		return false
	}
	for _, r := range roles {
		if p.Role == r {
			return true
		}
	}
	return false
}
