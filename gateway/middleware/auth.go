package middleware

import (
	"net/http"
	"strings"

	"github.com/dimasd-angga/go-microkit/internal/auth"
	apperr "github.com/dimasd-angga/go-microkit/internal/errors"
	"github.com/dimasd-angga/go-microkit/internal/httpx"
)

func RequireAuth(signer *auth.Signer) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tok := bearer(r)
			if tok == "" {
				httpx.Error(w, apperr.Unauthorized("missing bearer token"))
				return
			}
			claims, err := signer.Verify(tok)
			if err != nil {
				httpx.Error(w, apperr.Unauthorized("invalid or expired token"))
				return
			}
			ctx := auth.WithPrincipal(r.Context(), auth.Principal{
				UserID: claims.UserID,
				Email:  claims.Email,
				Role:   claims.Role,
			})
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func RequireRole(roles ...auth.Role) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !auth.RequireRole(r.Context(), roles...) {
				httpx.Error(w, apperr.Forbidden("insufficient role"))
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func bearer(r *http.Request) string {
	h := r.Header.Get("Authorization")
	if !strings.HasPrefix(h, "Bearer ") {
		return ""
	}
	return strings.TrimSpace(strings.TrimPrefix(h, "Bearer "))
}
