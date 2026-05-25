# go-microkit 🚀

> Production-ready Go microservices starter kit — clone it, run `make up`, and you're live with three services, gRPC, auth, messaging, distributed tracing, and a full observability stack. Zero boilerplate left for you to wire.

[![Go Version](https://img.shields.io/badge/Go-1.22+-00ADD8?style=flat&logo=go)](https://golang.org)
[![License: MIT](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)
[![Docker](https://img.shields.io/badge/Docker-ready-2496ED?style=flat&logo=docker)](https://docker.com)
[![gRPC](https://img.shields.io/badge/gRPC-enabled-244c5a?style=flat)](https://grpc.io)
[![OpenTelemetry](https://img.shields.io/badge/OpenTelemetry-traced-f5a800?style=flat)](https://opentelemetry.io)
[![PRs Welcome](https://img.shields.io/badge/PRs-welcome-brightgreen.svg)](CONTRIBUTING.md)

---

## Why another Go microservices kit?

Most Go starter kits are one of two things: too bare (a single service, no messaging, nothing shared) or too magical (a framework that hides Go from you). This kit takes a third path:

- **Real architecture, real services.** Three services that actually talk to each other via gRPC and async events — not toy handlers.
- **Shared `internal/` packages done right.** Logger, tracer, config, typed errors, and test utilities — wired identically across every service so new services just work.
- **One command to run everything.** `make up` starts Postgres, Redis, NATS, three services, Prometheus, Grafana, and Jaeger. No manual config, no missing env vars.
- **Opinionated but not a framework.** Clean architecture per service, no code generation magic, no DSLs. Just Go.

---

## Features

| Category | What's included |
|---|---|
| **Services** | User · Order · Notification — each with clean architecture (handler → usecase → repo) |
| **Communication** | gRPC (inter-service) + REST (external via API gateway) |
| **Messaging** | NATS JetStream for async event-driven flows |
| **Auth** | JWT access + refresh tokens, RBAC middleware |
| **Database** | PostgreSQL with `pgx`, schema migrations via `golang-migrate` |
| **Caching** | Redis for session store and hot-path caching |
| **Observability** | OpenTelemetry traces → Jaeger, Prometheus metrics → Grafana dashboards |
| **Logging** | Structured JSON logging via `zerolog`, trace-correlated |
| **Config** | Env-based config with validation (no YAML magic) |
| **Testing** | Integration test helpers, mock DB seeding, testcontainers |
| **DevOps** | Docker Compose, Makefile, GitHub Actions CI |

---

## Quick start

**Prerequisites:** Go 1.22+, Docker, Make

```bash
# 1. Clone
git clone https://github.com/dimasd-angga/go-microkit.git
cd go-microkit

# 2. Start everything
make up

# 3. Verify
curl http://localhost:8080/health
# → {"status":"ok","services":["user","order","notification"]}

# 4. Create a user
curl -X POST http://localhost:8080/api/v1/users \
  -H "Content-Type: application/json" \
  -d '{"email":"dimas@example.com","name":"Dimas","password":"supersecret"}'
```

That's it. Grafana is at `http://localhost:3000`, Jaeger at `http://localhost:16686`.

---

## Project structure

```
go-microkit/
├── services/
│   ├── user/                   # User service
│   │   ├── handler/            # HTTP + gRPC handlers
│   │   ├── usecase/            # Business logic (interface-driven)
│   │   ├── repository/         # Postgres implementation
│   │   ├── entity/             # Domain entities
│   │   └── main.go
│   ├── order/                  # Order service (same structure)
│   └── notification/           # Notification service (email, push, webhook)
│
├── gateway/                    # API gateway
│   ├── middleware/             # Auth, rate-limit, request-id
│   ├── router/                 # Route registration
│   └── main.go
│
├── internal/                   # Shared packages (not importable externally)
│   ├── auth/                   # JWT sign/verify, RBAC helpers
│   ├── config/                 # Env config loader with validation
│   ├── errors/                 # Typed domain errors (NotFound, Unauthorized…)
│   ├── logger/                 # zerolog wrapper with trace correlation
│   ├── tracer/                 # OpenTelemetry setup (OTLP exporter)
│   ├── messaging/              # NATS JetStream publisher/subscriber helpers
│   ├── grpcx/                  # gRPC server/dial helpers, error mapping
│   ├── httpx/                  # JSON response + error helpers
│   └── testutil/               # Test DB seeding, HTTP test helpers
│
├── proto/                      # Protobuf definitions + generated code
│   ├── user/user.proto
│   └── order/order.proto
│
├── deploy/
│   ├── docker-compose.yml              # Core services
│   ├── docker-compose.observability.yml # Prometheus, Grafana, Jaeger
│   └── k8s/                            # Helm chart (basic, production-ready)
│
├── scripts/
│   ├── gen-proto.sh            # Regenerate gRPC stubs
│   └── seed.sql                # Dev seed data
│
├── Makefile                    # All commands: up, down, test, proto, lint…
└── .github/
    └── workflows/
        └── ci.yml              # Lint + test on every PR
```

---

## Architecture

```
Client
  │
  ▼
API Gateway  ──── JWT middleware, rate limiting, routing
  │
  ├──── gRPC ──▶  User Service    ──▶  PostgreSQL
  │                    │
  ├──── gRPC ──▶  Order Service   ──▶  PostgreSQL
  │                    │
  │              NATS JetStream ◀─┘  (async events)
  │                    │
  └──── gRPC ──▶  Notification Service ──▶ Redis
                       │
              All services export traces → Jaeger
              All services export metrics → Prometheus → Grafana
```

---

## Makefile commands

```bash
make up           # Start all services + infra (Docker Compose)
make down         # Stop and remove containers
make test         # Run all tests (unit + integration)
make test-unit    # Unit tests only (fast)
make lint         # golangci-lint
make proto        # Regenerate gRPC code from .proto files
make migrate-up   # Run pending DB migrations
make migrate-down # Rollback last migration
make seed         # Seed dev data
make logs         # Follow all service logs
```

---

## Service deep dive

### User service

Handles registration, login, JWT issuance, and RBAC.

```go
// Clean architecture: handler knows nothing about DB
type UserUsecase interface {
    Register(ctx context.Context, in RegisterInput) (*entity.User, error)
    Login(ctx context.Context, email, password string) (*LoginResult, error)
    GetByID(ctx context.Context, id uuid.UUID) (*entity.User, error)
}
```

The repository layer is an interface — swap Postgres for anything else with zero usecase changes.

### Order service

Publishes `orders.created` and `orders.paid` events to NATS JetStream after every mutation. The notification service subscribes and sends the right comms without the order service knowing anything about notifications.

```go
// Publishing an event — one line from the usecase
err = u.pub.Publish(ctx, messaging.SubjOrderCreated, OrderCreatedEvent{
    OrderID:    o.ID,
    UserID:     o.UserID,
    TotalCents: o.TotalCents,
})
```

### Shared `internal/` packages

Every service imports these the same way:

```go
import (
    "github.com/dimasd-angga/go-microkit/internal/logger"
    "github.com/dimasd-angga/go-microkit/internal/tracer"
    apperr "github.com/dimasd-angga/go-microkit/internal/errors"
)

// Structured, trace-correlated log
logger.Info(ctx, "order created", "order_id", order.ID)

// Typed errors — the gateway maps these to HTTP status automatically
return nil, apperr.NotFound("user", userID)
```

---

## Configuration

All config is environment-based. Copy `.env.example` to `.env` and adjust:

```env
# Service ports
USER_SERVICE_GRPC_PORT=50051
ORDER_SERVICE_GRPC_PORT=50052
GATEWAY_PORT=8080

# Database
POSTGRES_DSN=postgres://microkit:microkit@localhost:5432/microkit?sslmode=disable

# Redis
REDIS_ADDR=localhost:6379

# NATS
NATS_URL=nats://localhost:4222

# Auth
JWT_SECRET=change-me-in-production
JWT_ACCESS_TTL=15m
JWT_REFRESH_TTL=168h

# Telemetry
OTEL_EXPORTER_OTLP_ENDPOINT=localhost:4317
```

No YAML, no Viper, no surprises. Validation on startup — missing required vars crash early with a clear message.

---

## Testing

```bash
# Run all tests
make test

# Unit tests (no Docker needed)
make test-unit

# Integration tests (requires Docker for testcontainers)
make test-integration
```

Integration tests run against a real Postgres provided by docker-compose. No mocks for the DB layer, no flaky state.

```go
// testutil makes this one line
db := testutil.NewTestDB(t)         // pool against POSTGRES_TEST_DSN
testutil.ApplyMigrations(t, db, "services/user/migrations")
```

---

## Observability

After `make up`, open:

| Tool | URL | What you see |
|---|---|---|
| **Grafana** | http://localhost:3000 | Service dashboards (RPS, latency, error rate) |
| **Jaeger** | http://localhost:16686 | Distributed traces across all services |
| **Prometheus** | http://localhost:9090 | Raw metrics scrape |

Every HTTP request and gRPC call is automatically traced end-to-end. No manual instrumentation needed in business logic — the middleware handles it.

---

## Deploying to production

A basic Helm chart is included in `deploy/k8s/`. It covers:

- Deployments with resource limits
- ConfigMaps for env vars
- Secrets for sensitive config
- HorizontalPodAutoscaler for each service
- Ingress with TLS termination

```bash
helm install go-microkit ./deploy/k8s \
  --set global.image.tag=v0.1.0 \
  --set postgres.externalDSN="postgres://..." \
  --set jwt.secret="..."
```

---

## Adding a new service

1. Copy `services/user/` → `services/yourservice/`
2. Rename packages and update imports
3. Define your `.proto` in `proto/yourservice/`, run `make proto`
4. Add your service to `deploy/docker-compose.yml`
5. Register routes in `gateway/router/router.go`

The `internal/` packages plug in automatically — logger, tracer, config, and typed errors are already there.

---

## Roadmap

- [ ] Rate limiting per user (Redis sliding window)
- [ ] GraphQL gateway option (alongside REST)
- [ ] Service mesh example (Istio sidecar config)
- [ ] gRPC reflection + Evans CLI integration guide
- [ ] Kafka alternative to NATS (pluggable messaging interface)
- [ ] OpenAPI spec auto-generated from proto

---

## Contributing

Contributions are welcome. Please read [CONTRIBUTING.md](CONTRIBUTING.md) first.

Bug reports, feature requests, and PRs are all appreciated. If this kit helped you ship faster, a ⭐ star means a lot.

---

## License

MIT © [Dimas Darfi Angga](https://github.com/dimasd-angga) — see [LICENSE](LICENSE)

---

## Related projects

- [golang-standards/project-layout](https://github.com/golang-standards/project-layout) — canonical Go project structure
- [google/wire](https://github.com/google/wire) — compile-time DI (optional addition to this kit)
- [grpc-ecosystem/grpc-gateway](https://github.com/grpc-ecosystem/grpc-gateway) — REST-to-gRPC transcoding
- [uber-go/fx](https://github.com/uber-go/fx) — dependency injection framework

<!-- go microservices, golang microservices starter kit, go grpc microservices, golang clean architecture, go microservices boilerplate, golang docker compose, go opentelemetry, golang observability, go nats jetstream, golang jwt auth, microservices golang example, go backend starter, golang production template -->
