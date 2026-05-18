.PHONY: help up down logs build test test-unit test-integration lint proto migrate-up migrate-down seed clean tidy

SHELL := /bin/bash
COMPOSE := docker compose -f deploy/docker-compose.yml -f deploy/docker-compose.observability.yml

help:
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-22s\033[0m %s\n", $$1, $$2}'

up: ## start all services + infra
	$(COMPOSE) up -d --build
	@echo ""
	@echo "  gateway:    http://localhost:8080"
	@echo "  grafana:    http://localhost:3000  (admin/admin)"
	@echo "  jaeger:     http://localhost:16686"
	@echo "  prometheus: http://localhost:9090"
	@echo ""

down: ## stop everything
	$(COMPOSE) down

logs: ## tail all logs
	$(COMPOSE) logs -f --tail=200

build: ## build all binaries to ./bin
	@mkdir -p bin
	go build -o bin/gateway ./gateway
	go build -o bin/user ./services/user
	go build -o bin/order ./services/order
	go build -o bin/notification ./services/notification

test: test-unit test-integration ## run all tests

test-unit:
	go test -count=1 -race -short ./...

test-integration:
	go test -count=1 -race -tags=integration ./...

lint:
	golangci-lint run --timeout=3m

proto:
	./scripts/gen-proto.sh

migrate-up:
	@for svc in user order; do \
		echo ">> migrating $$svc"; \
		migrate -path ./services/$$svc/migrations -database "$$POSTGRES_DSN" up; \
	done

migrate-down:
	@for svc in user order; do \
		migrate -path ./services/$$svc/migrations -database "$$POSTGRES_DSN" down 1; \
	done

seed:
	psql "$$POSTGRES_DSN" -f scripts/seed.sql

tidy:
	go mod tidy

clean:
	rm -rf bin/ coverage.out coverage.html
