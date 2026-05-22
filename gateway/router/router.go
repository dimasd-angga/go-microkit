package router

import (
	"context"
	"net/http"
	"time"

	"github.com/dimasd-angga/go-microkit/gateway/middleware"
	"github.com/dimasd-angga/go-microkit/internal/auth"
	apperr "github.com/dimasd-angga/go-microkit/internal/errors"
	"github.com/dimasd-angga/go-microkit/internal/httpx"
	orderpb "github.com/dimasd-angga/go-microkit/proto/order"
	userpb "github.com/dimasd-angga/go-microkit/proto/user"
	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/google/uuid"
)

type Deps struct {
	Signer      *auth.Signer
	UserClient  userpb.UserServiceClient
	OrderClient orderpb.OrderServiceClient
}

func New(d Deps) http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.Logging)
	r.Use(chimw.Recoverer)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-Request-ID"},
		ExposedHeaders:   []string{"X-Request-ID"},
		AllowCredentials: false,
		MaxAge:           300,
	}))

	rl := middleware.NewRateLimiter(100, 50)
	r.Use(rl.Handler())

	r.Get("/health", health(d))

	r.Route("/api/v1", func(r chi.Router) {
		r.Post("/users", registerHandler(d))
		r.Post("/auth/login", loginHandler(d))

		r.Group(func(r chi.Router) {
			r.Use(middleware.RequireAuth(d.Signer))

			r.Get("/me", meHandler(d))

			r.Post("/orders", createOrderHandler(d))
			r.Get("/orders/{id}", getOrderHandler(d))
			r.Get("/orders", listOrdersHandler(d))
			r.Post("/orders/{id}/pay", payOrderHandler(d))
		})
	})

	return r
}

// ----- handlers -----

func health(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		httpx.JSON(w, http.StatusOK, map[string]any{
			"status":   "ok",
			"services": []string{"user", "order", "notification"},
			"time":     time.Now().UTC().Format(time.RFC3339),
		})
	}
}

type registerReq struct {
	Email    string `json:"email"`
	Name     string `json:"name"`
	Password string `json:"password"`
}

func registerHandler(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var in registerReq
		if err := httpx.DecodeJSON(r, &in, 0); err != nil {
			httpx.Error(w, apperr.Invalid("bad_json", err.Error()))
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		resp, err := d.UserClient.Register(ctx, &userpb.RegisterRequest{
			Email: in.Email, Name: in.Name, Password: in.Password,
		})
		if err != nil {
			httpx.Error(w, fromGRPC(err))
			return
		}
		httpx.JSON(w, http.StatusCreated, resp)
	}
}

type loginReq struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func loginHandler(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var in loginReq
		if err := httpx.DecodeJSON(r, &in, 0); err != nil {
			httpx.Error(w, apperr.Invalid("bad_json", err.Error()))
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		resp, err := d.UserClient.Login(ctx, &userpb.LoginRequest{
			Email: in.Email, Password: in.Password,
		})
		if err != nil {
			httpx.Error(w, fromGRPC(err))
			return
		}
		httpx.JSON(w, http.StatusOK, resp)
	}
}

func meHandler(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		p, ok := auth.PrincipalFrom(r.Context())
		if !ok {
			httpx.Error(w, apperr.Unauthorized(""))
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		resp, err := d.UserClient.GetByID(ctx, &userpb.GetByIDRequest{Id: p.UserID.String()})
		if err != nil {
			httpx.Error(w, fromGRPC(err))
			return
		}
		httpx.JSON(w, http.StatusOK, resp)
	}
}

type createOrderReq struct {
	Items []struct {
		SKU       string `json:"sku"`
		Quantity  int32  `json:"quantity"`
		UnitCents int64  `json:"unit_cents"`
	} `json:"items"`
}

func createOrderHandler(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		p, _ := auth.PrincipalFrom(r.Context())
		var in createOrderReq
		if err := httpx.DecodeJSON(r, &in, 0); err != nil {
			httpx.Error(w, apperr.Invalid("bad_json", err.Error()))
			return
		}
		items := make([]*orderpb.OrderItem, 0, len(in.Items))
		for _, it := range in.Items {
			items = append(items, &orderpb.OrderItem{
				Sku: it.SKU, Quantity: it.Quantity, UnitCents: it.UnitCents,
			})
		}

		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()
		resp, err := d.OrderClient.Create(ctx, &orderpb.CreateOrderRequest{
			UserId: p.UserID.String(),
			Items:  items,
		})
		if err != nil {
			httpx.Error(w, fromGRPC(err))
			return
		}
		httpx.JSON(w, http.StatusCreated, resp)
	}
}

func getOrderHandler(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		if _, err := uuid.Parse(id); err != nil {
			httpx.Error(w, apperr.Invalid("invalid_id", "id must be a UUID"))
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()
		resp, err := d.OrderClient.GetByID(ctx, &orderpb.GetByIDRequest{Id: id})
		if err != nil {
			httpx.Error(w, fromGRPC(err))
			return
		}
		// Naive ownership check — replace with proper authz when adding admins.
		p, _ := auth.PrincipalFrom(r.Context())
		if resp.GetUserId() != p.UserID.String() && p.Role != auth.RoleAdmin {
			httpx.Error(w, apperr.Forbidden("not your order"))
			return
		}
		httpx.JSON(w, http.StatusOK, resp)
	}
}

func listOrdersHandler(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		p, _ := auth.PrincipalFrom(r.Context())
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()
		resp, err := d.OrderClient.ListByUser(ctx, &orderpb.ListByUserRequest{
			UserId: p.UserID.String(),
			Limit:  20,
			Offset: 0,
		})
		if err != nil {
			httpx.Error(w, fromGRPC(err))
			return
		}
		httpx.JSON(w, http.StatusOK, resp)
	}
}

func payOrderHandler(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()
		resp, err := d.OrderClient.MarkPaid(ctx, &orderpb.MarkPaidRequest{Id: id})
		if err != nil {
			httpx.Error(w, fromGRPC(err))
			return
		}
		httpx.JSON(w, http.StatusOK, resp)
	}
}
